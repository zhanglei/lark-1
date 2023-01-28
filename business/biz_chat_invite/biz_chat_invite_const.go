package biz_chat_invite

import "errors"

const (
	ERROR_CODE_CHAT_INVITE_GRPC_SERVICE_FAILURE    int32 = 10601
	ERROR_CODE_CHAT_INVITE_GET_INVITER_INFO_FAILED int32 = 10602
	ERROR_CODE_CHAT_INVITE_GET_INVITEE_INFO_FAILED int32 = 10602
	ERROR_CODE_CHAT_INVITE_GET_CHAT_INFO_FAILED    int32 = 10604
	ERROR_CODE_CHAT_INVITE_GET_USER_INFO_FAILED    int32 = 10605
)

const (
	ERROR_CHAT_INVITE_GRPC_SERVICE_FAILURE    = "服务故障"
	ERROR_CHAT_INVITE_GET_INVITER_INFO_FAILED = "获取邀请人信息失败"
	ERROR_CHAT_INVITE_GET_INVITEE_INFO_FAILED = "获取被邀请人信息失败"
	ERROR_CHAT_INVITE_GET_CHAT_INFO_FAILED    = "获取Chat信息失败"
	ERROR_CHAT_INVITE_GET_USER_INFO_FAILED    = "获取人员信息失败"
)

var (
	ERR_CHAT_INVITE_QUERY_DB_FAILED = errors.New("查询失败")
)
