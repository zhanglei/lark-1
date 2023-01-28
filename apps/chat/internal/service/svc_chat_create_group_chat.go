package service

import (
	"context"
	"fmt"
	"gorm.io/gorm"
	"lark/business/biz_chat_invite"
	"lark/domain/po"
	"lark/pkg/common/xants"
	"lark/pkg/common/xlog"
	"lark/pkg/common/xmysql"
	"lark/pkg/common/xsnowflake"
	"lark/pkg/constant"
	"lark/pkg/entity"
	"lark/pkg/proto/pb_chat"
	"lark/pkg/proto/pb_dist"
	"lark/pkg/proto/pb_enum"
	"lark/pkg/proto/pb_invite"
)

func (s *chatService) CreateGroupChat(ctx context.Context, req *pb_chat.CreateGroupChatReq) (resp *pb_chat.CreateGroupChatResp, _ error) {
	resp = &pb_chat.CreateGroupChatResp{}
	var (
		creator *po.User
		tx      *gorm.DB
		w       = entity.NewMysqlWhere()
		chat    *po.Chat
		err     error
	)
	var (
		avatar        *po.Avatar
		member        *po.ChatMember
		invitationMsg string
		uid           int64
		invite        *po.ChatInvite
		inviteList    = make([]*po.ChatInvite, 0)
	)

	// 1 获取创建者信息
	w.SetFilter("uid=?", req.CreatorUid)
	creator, err = s.userRepo.UserInfo(w)
	if err != nil {
		resp.Set(ERROR_CODE_CHAT_QUERY_DB_FAILED, ERROR_CHAT_QUERY_DB_FAILED)
		xlog.Warn(ERROR_CODE_CHAT_QUERY_DB_FAILED, ERROR_CHAT_QUERY_DB_FAILED, err.Error())
		return
	}

	// 2 构建chat模型
	chat = &po.Chat{
		CreatorUid: req.CreatorUid,
		ChatType:   int(pb_enum.CHAT_TYPE_GROUP),
		AvatarKey:  constant.CONST_AVATAR_KEY_SMALL,
		Name:       req.Name,
		About:      req.About,
	}
	tx = xmysql.GetTX()
	defer func() {
		if err == nil {
			tx.Commit()
		} else {
			tx.Rollback()
		}
	}()

	// 3 chat入库
	err = s.chatRepo.TxCreate(tx, chat)
	if err != nil {
		resp.Set(ERROR_CODE_CHAT_INSERT_VALUE_FAILED, ERROR_CHAT_INSERT_VALUE_FAILED)
		xlog.Warn(ERROR_CODE_CHAT_INSERT_VALUE_FAILED, ERROR_CHAT_INSERT_VALUE_FAILED, err.Error())
		return
	}

	// 4 creator入群/入库
	member = &po.ChatMember{
		ChatId:          chat.ChatId,
		ChatType:        chat.ChatType,
		ChatName:        chat.Name,
		Uid:             creator.Uid,
		RoleId:          int(pb_enum.CHAT_GROUP_ROLE_MASTER),
		Alias:           creator.Nickname,
		MemberAvatarKey: creator.AvatarKey,
		ChatAvatarKey:   chat.AvatarKey,
		Sync:            constant.SYNCHRONIZE_USER_INFO,
		ServerId:        creator.ServerId,
	}
	err = s.chatMemberRepo.TxCreate(tx, member)
	if err != nil {
		resp.Set(ERROR_CODE_CHAT_INSERT_VALUE_FAILED, ERROR_CHAT_INSERT_VALUE_FAILED)
		xlog.Warn(ERROR_CODE_CHAT_INSERT_VALUE_FAILED, ERROR_CHAT_INSERT_VALUE_FAILED, err.Error())
		return
	}

	// 5 设置群头像
	avatar = &po.Avatar{
		OwnerId:      chat.ChatId,
		OwnerType:    int(pb_enum.AVATAR_OWNER_CHAT_AVATAR),
		AvatarSmall:  constant.CONST_AVATAR_KEY_SMALL,
		AvatarMedium: constant.CONST_AVATAR_KEY_MEDIUM,
		AvatarLarge:  constant.CONST_AVATAR_KEY_LARGE,
	}
	err = s.avatarRepo.TxCreate(tx, avatar)
	if err != nil {
		resp.Set(ERROR_CODE_CHAT_INSERT_VALUE_FAILED, ERROR_CHAT_INSERT_VALUE_FAILED)
		xlog.Warn(ERROR_CODE_CHAT_INSERT_VALUE_FAILED, ERROR_CHAT_INSERT_VALUE_FAILED, err.Error())
		return
	}

	// 6 缓存成员信息
	err = s.chatMemberCache.HSetNXChatMember(member.ChatId, member.Uid, fmt.Sprintf("%d,%d,%d", member.ServerId, member.Uid, member.Status))
	if err != nil {
		resp.Set(ERROR_CODE_CHAT_CACHE_CHAT_MEMBER_FAILED, ERROR_CHAT_CACHE_CHAT_MEMBER_FAILED)
		xlog.Warn(ERROR_CODE_CHAT_CACHE_CHAT_MEMBER_FAILED, ERROR_CHAT_CACHE_CHAT_MEMBER_FAILED, err.Error())
		return
	}

	// 7 构建邀请信息
	invitationMsg = creator.Nickname + CONST_CHAT_INVITE_TITLE_CONJUNCTION + chat.Name
	for _, uid = range req.UidList {
		if uid == req.CreatorUid {
			continue
		}
		invite = &po.ChatInvite{
			InviteId:      xsnowflake.NewSnowflakeID(),
			ChatId:        chat.ChatId,
			ChatType:      chat.ChatType,
			InitiatorUid:  req.CreatorUid,
			InviteeUid:    uid,
			InvitationMsg: invitationMsg,
		}
		inviteList = append(inviteList, invite)
	}
	if len(inviteList) == 0 {
		return
	}

	// 8 邀请信息入库
	err = s.chatInviteRepo.TxNewChatInviteList(tx, inviteList)
	if err != nil {
		resp.Set(ERROR_CODE_CHAT_INSERT_VALUE_FAILED, ERROR_CHAT_INSERT_VALUE_FAILED)
		xlog.Warn(ERROR_CODE_CHAT_INSERT_VALUE_FAILED, ERROR_CHAT_INSERT_VALUE_FAILED, err.Error())
		return
	}

	xants.Submit(func() {
		// 9 缓存
		s.cacheChatInfo(chat)
		// 10 邀请推送
		inviteReq := &pb_invite.InitiateChatInviteReq{
			ChatId:        chat.ChatId,
			ChatType:      pb_enum.CHAT_TYPE(chat.ChatType),
			InitiatorUid:  req.CreatorUid,
			InviteeUids:   req.UidList,
			InvitationMsg: invitationMsg,
			Platform:      0,
		}
		s.sendChatInviteNotificationMessage(inviteReq, inviteList)
	})
	return
}

func (s *chatService) sendChatInviteNotificationMessage(inviteReq *pb_invite.InitiateChatInviteReq, invitees []*po.ChatInvite) {
	var (
		req  *pb_dist.ChatInviteNotificationReq
		resp *pb_dist.ChatInviteNotificationResp
		err  error
	)
	req, err = biz_chat_invite.ConstructChatInviteNotificationMessage(
		inviteReq,
		invitees,
		s.cfg.Redis.Prefix,
		s.chatCache,
		s.userCache,
		s.chatClient,
		s.userClient)
	if err != nil {
		return
	}
	if req == nil {
		return
	}
	resp = s.distClient.ChatInviteNotification(req)
	if resp == nil {
		xlog.Warn(ERROR_CODE_CHAT_GRPC_SERVICE_FAILURE, ERROR_CHAT_GRPC_SERVICE_FAILURE)
		return
	}
	if resp.Code > 0 {
		xlog.Warn(resp.Code, resp.Msg)
	}
}
