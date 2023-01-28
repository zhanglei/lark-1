package service

import (
	"context"
	"fmt"
	"github.com/go-sql-driver/mysql"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
	"lark/domain/po"
	"lark/pkg/common/xants"
	"lark/pkg/common/xlog"
	"lark/pkg/common/xmysql"
	"lark/pkg/common/xsnowflake"
	"lark/pkg/constant"
	"lark/pkg/entity"
	"lark/pkg/proto/pb_chat_member"
	"lark/pkg/proto/pb_enum"
	"lark/pkg/proto/pb_invite"
	"lark/pkg/proto/pb_mq"
	"lark/pkg/proto/pb_msg"
	"lark/pkg/proto/pb_user"
	"lark/pkg/utils"
)

func (s *chatInviteService) ChatInviteHandle(ctx context.Context, req *pb_invite.ChatInviteHandleReq) (resp *pb_invite.ChatInviteHandleResp, _ error) {
	resp = new(pb_invite.ChatInviteHandleResp)
	var (
		tx     *gorm.DB
		u      = entity.NewMysqlUpdate()
		invite *po.ChatInvite
		cont   bool
		err    error
	)
	// 1 校验邀请
	invite, cont = s.chatInviteExists(req, resp)
	if cont == false {
		return
	}
	// 2 重复操作验证
	cont = s.alreadyMember(invite, resp)
	if cont == false {
		return
	}
	// 3 更新邀请
	u.SetFilter("invite_id=?", req.InviteId)
	u.Set("handler_uid", req.HandlerUid)
	u.Set("handle_result", req.HandleResult)
	u.Set("handle_msg", req.HandleMsg)
	u.Set("handled_ts", utils.NowMilli())

	tx = xmysql.GetTX()
	defer func() {
		if err == nil {
			tx.Commit()
		} else {
			tx.Rollback()
		}
	}()

	err = s.chatInviteRepo.TxUpdateChatInvite(tx, u)
	if err != nil {
		resp.Set(ERROR_CODE_CHAT_INVITE_UPDATE_VALUE_FAILED, ERROR_CHAT_INVITE_UPDATE_VALUE_FAILED)
		xlog.Warn(ERROR_CODE_CHAT_INVITE_UPDATE_VALUE_FAILED, ERROR_CHAT_INVITE_UPDATE_VALUE_FAILED, err.Error())
		return
	}
	if req.HandleResult == pb_enum.INVITE_HANDLE_RESULT_REFUSE {
		// 4 拒绝邀请
		return
	}
	// 5 同意邀请
	err = s.acceptInvitation(tx, invite, resp)
	return
}

func (s *chatInviteService) chatInviteExists(req *pb_invite.ChatInviteHandleReq, resp *pb_invite.ChatInviteHandleResp) (invite *po.ChatInvite, cont bool) {
	var (
		w   = entity.NewMysqlWhere()
		err error
	)
	// 1 校验邀请
	w.SetFilter("invite_id=?", req.InviteId)
	invite, err = s.chatInviteRepo.ChatInvite(w)
	if err != nil {
		resp.Set(ERROR_CODE_CHAT_INVITE_QUERY_DB_FAILED, ERROR_CHAT_INVITE_QUERY_DB_FAILED)
		xlog.Warn(ERROR_CODE_CHAT_INVITE_QUERY_DB_FAILED, ERROR_CHAT_INVITE_QUERY_DB_FAILED, err.Error())
		return
	}
	if invite.InviteId == 0 {
		resp.Set(ERROR_CODE_CHAT_INVITE_QUERY_DB_FAILED, ERROR_CHAT_INVITE_QUERY_DB_FAILED)
		xlog.Warn(ERROR_CODE_CHAT_INVITE_QUERY_DB_FAILED, ERROR_CHAT_INVITE_QUERY_DB_FAILED)
		return
	}
	if invite.HandleResult != 0 {
		resp.Set(ERROR_CODE_CHAT_INVITE_HAS_HANDLED, ERROR_CHAT_INVITE_HAS_HANDLED)
		xlog.Warn(ERROR_CODE_CHAT_INVITE_HAS_HANDLED, ERROR_CHAT_INVITE_HAS_HANDLED)
		return
	}
	if req.HandlerUid != invite.InviteeUid {
		resp.Set(ERROR_CODE_CHAT_INVITE_BAD_HANDLER, ERROR_CHAT_INVITE_BAD_HANDLER)
		xlog.Warn(ERROR_CODE_CHAT_INVITE_BAD_HANDLER, ERROR_CHAT_INVITE_BAD_HANDLER)
		return
	}
	cont = true
	return
}

func (s *chatInviteService) alreadyMember(invite *po.ChatInvite, resp *pb_invite.ChatInviteHandleResp) (cont bool) {
	var (
		w     = entity.NewMysqlWhere()
		count int64
		err   error
	)
	// 2 重复操作验证
	w.SetFilter("chat_id=?", invite.InviteId)
	w.SetFilter("uid = ?", invite.InviteeUid)
	count, err = s.chatMemberRepo.ChatMemberCount(w)
	if err != nil {
		resp.Set(ERROR_CODE_CHAT_INVITE_QUERY_DB_FAILED, ERROR_CHAT_INVITE_QUERY_DB_FAILED)
		xlog.Warn(ERROR_CODE_CHAT_INVITE_QUERY_DB_FAILED, ERROR_CHAT_INVITE_QUERY_DB_FAILED, err.Error())
		return
	}
	if count > 0 {
		//resp.Set(ERROR_CODE_CHAT_INVITE_ALREADY_MEMBER, ERROR_CHAT_INVITE_ALREADY_MEMBER)
		return
	}
	cont = true
	return
}

func (s *chatInviteService) acceptInvitation(tx *gorm.DB, invite *po.ChatInvite, resp *pb_invite.ChatInviteHandleResp) (err error) {
	// 5 同意邀请
	var (
		w           = entity.NewMysqlWhere()
		chat        *po.Chat
		members     []*po.ChatMember
		member      *po.ChatMember
		memberCount int
		list        []*pb_user.UserSrvInfo
		user        *pb_user.UserSrvInfo
		index       int
		uidList     []int64
		distMaps    map[string]interface{}
	)

	switch pb_enum.CHAT_TYPE(invite.ChatType) {
	case pb_enum.CHAT_TYPE_PRIVATE:
		// 6 创建Chat
		chat = &po.Chat{
			ChatId:     invite.ChatId,
			CreatorUid: invite.InitiatorUid,
			ChatType:   int(pb_enum.CHAT_TYPE_PRIVATE),
		}
		err = s.chatRepo.TxCreate(tx, chat)
		if err != nil {
			xlog.Warn(ERROR_CODE_CHAT_INVITE_INSERT_VALUE_FAILED, ERROR_CHAT_INVITE_INSERT_VALUE_FAILED, err.Error())
			switch err.(type) {
			case *mysql.MySQLError:
				if err.(*mysql.MySQLError).Number == constant.ERROR_CODE_MYSQL_DUPLICATE_ENTRY {
					err = nil
					resp.Set(ERROR_CODE_CHAT_INVITE_PROCESSED, ERROR_CHAT_INVITE_PROCESSED)
					return
				}
			}
			resp.Set(ERROR_CODE_CHAT_INVITE_INSERT_VALUE_FAILED, ERROR_CHAT_INVITE_INSERT_VALUE_FAILED)
			return
		}
		uidList = []int64{invite.InitiatorUid, invite.InviteeUid}
	case pb_enum.CHAT_TYPE_GROUP:
		w.SetFilter("chat_id=?", invite.ChatId)
		chat, err = s.chatRepo.TxChat(tx, w)
		if err != nil {
			resp.Set(ERROR_CODE_CHAT_INVITE_QUERY_DB_FAILED, ERROR_CHAT_INVITE_QUERY_DB_FAILED)
			xlog.Warn(ERROR_CODE_CHAT_INVITE_QUERY_DB_FAILED, ERROR_CHAT_INVITE_QUERY_DB_FAILED, err.Error())
			return
		}
		uidList = []int64{invite.InviteeUid}
	}

	memberCount = len(uidList)
	w.Reset()
	w.SetFilter("uid IN(?)", uidList)
	list, err = s.userRepo.TxUserSrvList(tx, w)
	if err != nil {
		resp.Set(ERROR_CODE_CHAT_INVITE_QUERY_DB_FAILED, ERROR_CHAT_INVITE_QUERY_DB_FAILED)
		xlog.Warn(ERROR_CODE_CHAT_INVITE_QUERY_DB_FAILED, ERROR_CHAT_INVITE_QUERY_DB_FAILED, err.Error())
		return
	}
	if len(list) != memberCount {
		err = ERR_CHAT_INVITE_QUERY_DB_FAILED
		resp.Set(ERROR_CODE_CHAT_INVITE_QUERY_DB_FAILED, ERROR_CHAT_INVITE_QUERY_DB_FAILED)
		xlog.Warn(ERROR_CODE_CHAT_INVITE_QUERY_DB_FAILED, ERROR_CHAT_INVITE_QUERY_DB_FAILED)
		return
	}
	members = make([]*po.ChatMember, memberCount)
	for index, user = range list {
		switch pb_enum.CHAT_TYPE(invite.ChatType) {
		case pb_enum.CHAT_TYPE_PRIVATE:
			var (
				info *pb_user.UserSrvInfo
			)
			if index == 0 {
				info = list[1]
			} else {
				info = list[0]
			}
			member = &po.ChatMember{
				ChatId:          invite.ChatId,
				ChatType:        invite.ChatType,
				Uid:             info.Uid,
				OwnerId:         user.Uid,
				Alias:           info.Nickname,
				MemberAvatarKey: info.AvatarKey,
				Sync:            constant.SYNCHRONIZE_USER_INFO,
				ServerId:        info.ServerId,
			}
		case pb_enum.CHAT_TYPE_GROUP:
			member = &po.ChatMember{
				ChatId:          invite.ChatId,
				ChatType:        invite.ChatType,
				Uid:             user.Uid,
				Alias:           user.Nickname,
				MemberAvatarKey: user.AvatarKey,
				Sync:            constant.SYNCHRONIZE_USER_INFO,
				ServerId:        user.ServerId,
				ChatAvatarKey:   chat.AvatarKey,
				ChatName:        chat.Name,
			}
		}
		members[index] = member
	}
	// 7 成为 chat member
	err = s.chatMemberRepo.TxCreateMultiple(tx, members)
	if err != nil {
		resp.Set(ERROR_CODE_CHAT_INVITE_INSERT_VALUE_FAILED, ERROR_CHAT_INVITE_INSERT_VALUE_FAILED)
		xlog.Warn(ERROR_CODE_CHAT_INVITE_INSERT_VALUE_FAILED, ERROR_CHAT_INVITE_INSERT_VALUE_FAILED, err.Error())
		return
	}
	distMaps = make(map[string]interface{})
	for _, member = range members {
		distMaps[utils.Int64ToStr(member.Uid)] = fmt.Sprintf("%d,%d,%d", member.ServerId, member.Uid, member.Status)
	}
	// 8 缓存 chat member
	err = s.chatMemberCache.HMSetChatMembers(member.ChatId, distMaps)
	if err != nil {
		resp.Set(ERROR_CODE_CHAT_INVITE_CACHE_CHAT_MEMBER_FAILED, ERROR_CHAT_INVITE_CACHE_CHAT_MEMBER_FAILED)
		xlog.Warn(ERROR_CODE_CHAT_INVITE_CACHE_CHAT_MEMBER_FAILED, ERROR_CHAT_INVITE_CACHE_CHAT_MEMBER_FAILED, err.Error())
		return
	}
	// 9 邀请成功推送
	xants.Submit(func() {
		s.chatInviteHandleMessage(chat, invite, members)
	})
	return
}

func (s *chatInviteService) chatInviteHandleMessage(chat *po.Chat, invite *po.ChatInvite, members []*po.ChatMember) {
	switch pb_enum.CHAT_TYPE(chat.ChatType) {
	case pb_enum.CHAT_TYPE_PRIVATE:
		if len(members) != 2 {
			return
		}
		//成为好友通知
		s.addedContactMessage(chat, invite, members)
	case pb_enum.CHAT_TYPE_GROUP:
		if len(members) != 1 {
			return
		}
		//加入群通知
		s.joinedChatGroupMessage(chat, invite, members[0])
	}
}

func (s *chatInviteService) addedContactMessage(chat *po.Chat, invite *po.ChatInvite, members []*po.ChatMember) {
	var (
		member   *po.ChatMember
		index    int
		m        *po.ChatMember
		seqId    int64
		nowMilli = utils.NowMilli()
		err      error
	)
	for index, member = range members {
		var (
			inbox *pb_mq.InboxMessage
		)
		if seqId, err = s.chatMessageCache.IncrSeqID(chat.ChatId); err != nil {
			return
		}
		if index == 0 {
			m = members[1]
		} else {
			m = members[0]
		}
		msg := &pb_msg.SrvChatMessage{
			SrvMsgId:        xsnowflake.NewSnowflakeID(),
			CliMsgId:        xsnowflake.NewSnowflakeID(),
			SenderId:        m.Uid,
			SenderPlatform:  0,
			SenderName:      m.Alias,
			SenderAvatarKey: m.MemberAvatarKey,
			ChatId:          chat.ChatId,
			ChatType:        pb_enum.CHAT_TYPE_PRIVATE,
			SeqId:           seqId,
			MsgFrom:         pb_enum.MSG_FROM_SYSTEM,
			MsgType:         0,
			Body:            nil,
			Status:          0,
			SentTs:          0,
			SrvTs:           0,
		}
		if member.OwnerId == chat.CreatorUid {
			msg.SentTs = nowMilli
			msg.SrvTs = nowMilli
			msg.Body = []byte(invite.InvitationMsg)
			msg.MsgType = pb_enum.MSG_TYPE_CHAT_INVITE_MSG
		} else {
			msg.SentTs = nowMilli + 1
			msg.SrvTs = nowMilli + 1
			msg.Body = []byte("I've accepted your friend request. Now let's chat!")
			msg.MsgType = pb_enum.MSG_TYPE_ACCEPTED_CHAT_INVITE
		}
		inbox = &pb_mq.InboxMessage{
			Topic:    pb_enum.TOPIC_CHAT,
			SubTopic: pb_enum.SUB_TOPIC_CHAT_MSG,
			Msg:      msg,
		}
		// 将消息推送到kafka消息队列
		_, _, err = s.producer.EnQueue(inbox, constant.CONST_MSG_KEY_MSG)
		if err != nil {
			xlog.Warn(ERROR_CODE_CHAT_INVITE_ENQUEUE_FAILED, ERROR_CHAT_INVITE_ENQUEUE_FAILED, err.Error())
			return
		}
	}
}

func (s *chatInviteService) joinedChatGroupMessage(chat *po.Chat, invite *po.ChatInvite, member *po.ChatMember) {
	var (
		initiator *pb_chat_member.ChatMemberInfo
		seqId     int64
		nowMilli  = utils.NowMilli()
		msg       *pb_msg.SrvChatMessage
		joinedMsg *pb_msg.JoinedGroupChatMessage
		inbox     *pb_mq.InboxMessage
		err       error
	)
	initiator, err = s.chatMemberCache.GetChatMemberInfo(invite.ChatId, invite.InitiatorUid)
	if err != nil {
		xlog.Warn(ERROR_CODE_CHAT_INVITE_REDIS_GET_FAILED, ERROR_CHAT_INVITE_REDIS_GET_FAILED, err.Error())
	}
	if initiator.Uid == 0 {
		w := entity.NewMysqlWhere()
		w.SetFilter("chat_id=?", invite.ChatId)
		w.SetFilter("uid=?", invite.InitiatorUid)
		initiator, err = s.chatMemberRepo.ChatMember(w)
		if err != nil {
			xlog.Warn(ERROR_CODE_CHAT_INVITE_QUERY_DB_FAILED, ERROR_CHAT_INVITE_QUERY_DB_FAILED, err.Error())
			return
		}
		if initiator.Uid == 0 {
			xlog.Warn(ERROR_CODE_CHAT_INVITE_QUERY_DB_FAILED, ERROR_CHAT_INVITE_QUERY_DB_FAILED)
			return
		}
	}

	if seqId, err = s.chatMessageCache.IncrSeqID(chat.ChatId); err != nil {
		xlog.Warn(ERROR_CODE_CHAT_INVITE_INCR_SEQ_ID_FAILED, ERROR_CHAT_INVITE_INCR_SEQ_ID_FAILED, err.Error())
		return
	}
	msg = &pb_msg.SrvChatMessage{
		SrvMsgId:        xsnowflake.NewSnowflakeID(),
		CliMsgId:        xsnowflake.NewSnowflakeID(),
		SenderId:        0,
		SenderPlatform:  0,
		SenderName:      "",
		SenderAvatarKey: "",
		ChatId:          chat.ChatId,
		ChatType:        pb_enum.CHAT_TYPE_GROUP,
		SeqId:           seqId,
		MsgFrom:         pb_enum.MSG_FROM_SYSTEM,
		MsgType:         pb_enum.MSG_TYPE_JOINED_GROUP_CHAT,
		Body:            nil,
		Status:          0,
		SentTs:          nowMilli,
		SrvTs:           nowMilli,
	}
	joinedMsg = &pb_msg.JoinedGroupChatMessage{
		Inviter: &pb_chat_member.ChatMemberBasicInfo{
			Uid:             initiator.Uid,
			Alias:           initiator.Alias,
			MemberAvatarKey: initiator.MemberAvatarKey,
		},
		Invitee: &pb_chat_member.ChatMemberBasicInfo{
			Uid:             member.Uid,
			Alias:           member.Alias,
			MemberAvatarKey: member.MemberAvatarKey,
		},
	}
	msg.Body, err = proto.Marshal(joinedMsg)
	if err != nil {
		xlog.Warn(ERROR_CODE_CHAT_INVITE_PROTOCOL_MARSHAL_ERR, ERROR_CHAT_INVITE_PROTOCOL_MARSHAL_ERR, err.Error())
		return
	}
	inbox = &pb_mq.InboxMessage{
		Topic:    pb_enum.TOPIC_CHAT,
		SubTopic: pb_enum.SUB_TOPIC_CHAT_JOINED_GROUP_CHAT,
		Msg:      msg,
	}
	// 将消息推送到kafka消息队列
	_, _, err = s.producer.EnQueue(inbox, constant.CONST_MSG_KEY_MSG)
	if err != nil {
		xlog.Warn(ERROR_CODE_CHAT_INVITE_ENQUEUE_FAILED, ERROR_CHAT_INVITE_ENQUEUE_FAILED, err.Error())
		return
	}
}
