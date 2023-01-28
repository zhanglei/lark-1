package service

import (
	"context"
	"github.com/jinzhu/copier"
	"lark/pkg/common/xlog"
	"lark/pkg/common/xmysql"
	"lark/pkg/constant"
	"lark/pkg/entity"
	"lark/pkg/proto/pb_enum"
	"lark/pkg/proto/pb_user"
	"lark/pkg/protocol"
)

func (s *userService) UploadAvatar(ctx context.Context, req *pb_user.UploadAvatarReq) (resp *pb_user.UploadAvatarResp, _ error) {
	resp = &pb_user.UploadAvatarResp{Avatar: &pb_user.AvatarInfo{}}
	var (
		u      = entity.NewMysqlUpdate()
		result *protocol.Result
		err    error
	)
	u.Set("avatar_small", req.AvatarSmall)
	u.Set("avatar_medium", req.AvatarMedium)
	u.Set("avatar_large", req.AvatarLarge)
	u.SetFilter("owner_id=?", req.OwnerId)
	u.SetFilter("owner_type=?", pb_enum.AVATAR_OWNER_USER_AVATAR)

	tx := xmysql.GetTX()
	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()
	err = s.avatarRepo.TxUpdateAvatar(tx, u)
	if err != nil {
		resp.Set(ERROR_CODE_USER_SET_AVATAR_FAILED, ERROR_USER_SET_AVATAR_FAILED)
		xlog.Warn(ERROR_CODE_USER_SET_AVATAR_FAILED, ERROR_USER_SET_AVATAR_FAILED, err.Error())
		return
	}

	u.Reset()
	u.SetFilter("uid=?", req.OwnerId)
	u.Set("avatar_key", req.AvatarSmall)
	err = s.userRepo.UpdateUser(u)
	if err != nil {
		resp.Set(ERROR_CODE_USER_SET_AVATAR_FAILED, ERROR_USER_SET_AVATAR_FAILED)
		xlog.Warn(ERROR_CODE_USER_SET_AVATAR_FAILED, ERROR_USER_SET_AVATAR_FAILED, err.Error())
		return
	}

	// 删除缓存
	err = s.userCache.DelUserInfo(s.cfg.Redis.Prefix, req.OwnerId)
	if err != nil {
		resp.Set(ERROR_CODE_USER_UPDATE_USER_CACHE_FAILED, ERROR_USER_UPDATE_USER_CACHE_FAILED)
		xlog.Warn(ERROR_CODE_USER_UPDATE_USER_CACHE_FAILED, ERROR_USER_UPDATE_USER_CACHE_FAILED, err.Error())
		return
	}

	u.Reset()
	u.SetFilter("sync=?", constant.SYNCHRONIZE_USER_INFO)
	u.SetFilter("uid=?", req.OwnerId)
	u.Set("member_avatar_key", req.AvatarSmall)
	err = s.chatMemberRepo.TxUpdateChatMember(tx, u)
	if err != nil {
		resp.Set(ERROR_CODE_USER_SET_AVATAR_FAILED, ERROR_USER_SET_AVATAR_FAILED)
		xlog.Warn(ERROR_CODE_USER_SET_AVATAR_FAILED, ERROR_USER_SET_AVATAR_FAILED, err.Error())
		return
	}

	result, err = s.updateChatMemberCacheInfo(tx, req.OwnerId)
	if err != nil {
		resp.Set(result.Code, result.Msg)
		return
	}
	copier.Copy(resp.Avatar, req)
	return
}
