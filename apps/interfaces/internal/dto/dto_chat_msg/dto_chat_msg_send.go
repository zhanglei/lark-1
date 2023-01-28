package dto_chat_msg

import "lark/pkg/proto/pb_enum"

type SendChatMessageReq struct {
	ChatId  int64            `json:"chat_id" validate:"required,gt=0"`        // 会话ID
	Body    string           `json:"body" validate:"required,min=1,max=1500"` // 消息本体
	MsgType pb_enum.MSG_TYPE `json:"msg_type" validate:"required,gt=0"`       // 消息类型
}
