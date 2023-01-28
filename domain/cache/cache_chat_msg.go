package cache

import (
	"github.com/go-redis/redis/v9"
	"github.com/jinzhu/copier"
	"lark/domain/po"
	"lark/pkg/common/xlog"
	"lark/pkg/common/xredis"
	"lark/pkg/constant"
	"lark/pkg/proto/pb_convo"
	"lark/pkg/utils"
)

const (
	CHAT_MESSAGE_REDIS_OPERATION_FAILED = "redis操作失败"
	CHAT_MESSAGE_EVALSHA_PARAM_ERROR    = "参数错误"
	CHAT_MESSAGE_REPEATED_MESSAGE       = "重复的消息"
)

type ChatMessageCache interface {
	GetChatMessage(chatId int64, seqId int64) (message *po.Message, err error)
	SetChatMessage(message *po.Message) (err error)
	SetConvoMessage(prefix string, message *po.Message) (err error)
	RepeatMessageVerify(prefix string, chatId int64, msgId int64) (r string, ok bool)
	IncrSeqID(chatId int64) (seqId int64, err error)
	GetMaxSeqID(chatId int64) (seqId uint64, err error)
	MGetMessages(keys ...string) ([]interface{}, error)
}

type chatMessageCache struct {
}

func NewChatMessageCache() ChatMessageCache {
	return &chatMessageCache{}
}

func (c *chatMessageCache) GetChatMessage(chatId int64, seqId int64) (message *po.Message, err error) {
	var (
		key = constant.RK_SYNC_MSG_CACHE + utils.Int64ToStr(chatId) + ":" + utils.Int64ToStr(seqId)
	)
	message = new(po.Message)
	err = Get(key, message)
	return
}

func (c *chatMessageCache) SetChatMessage(message *po.Message) (err error) {
	var (
		key = constant.RK_SYNC_MSG_CACHE + utils.Int64ToStr(message.ChatId) + ":" + utils.Int64ToStr(message.SeqId)
	)
	err = Set(key, message, constant.CONST_DURATION_MSG_CACHE_SECOND)
	return
}

func (c *chatMessageCache) SetConvoMessage(prefix string, message *po.Message) (err error) {
	var (
		key1 = prefix + constant.RK_SYNC_MSG_CACHE + utils.Int64ToStr(message.ChatId) + ":" + utils.Int64ToStr(message.SeqId)
		key2 = prefix + constant.RK_MSG_CONVO_MSG + utils.Int64ToStr(message.ChatId)
		cm   = new(pb_convo.ConvoMessage)
		val1 string
		val2 string
	)
	copier.Copy(cm, message)
	val1, err = utils.Marshal(message)
	if err != nil {
		xlog.Warn(ERROR_CODE_CACHE_PROTOCOL_MARSHAL_ERR, ERROR_CACHE_PROTOCOL_MARSHAL_ERR, err.Error())
		return
	}
	val2, err = utils.Marshal(cm)
	if err != nil {
		xlog.Warn(ERROR_CODE_CACHE_PROTOCOL_MARSHAL_ERR, ERROR_CACHE_PROTOCOL_MARSHAL_ERR, err.Error())
		return
	}
	err = xredis.EvalSha(xredis.SHA_MSET_CONVO_MESSAGE, []string{key1, key2}, []interface{}{constant.CONST_DURATION_SHA_CONVO_MESSAGE_SECOND, val1, val2})
	if err != nil {
		xlog.Warn(ERROR_CODE_CACHE_REDIS_SET_FAILED, ERROR_CACHE_REDIS_SET_FAILED, err.Error())
	}
	return
}

func (c *chatMessageCache) RepeatMessageVerify(prefix string, chatId int64, msgId int64) (r string, ok bool) {
	var (
		key    = prefix + constant.RK_MSG_CLI_MSG_ID + utils.Int64ToStr(chatId) + ":" + utils.Int64ToStr(msgId)
		result interface{}
		err    error
	)
	result, err = xredis.EvalShaResult(xredis.SHA_SET_MESSAGE_ID, []string{key}, []interface{}{constant.CONST_DURATION_SHA_MSG_ID_SECOND})
	if err != nil {
		r = CHAT_MESSAGE_REDIS_OPERATION_FAILED
		xlog.Warn(err.Error())
		return
	}
	switch result.(type) {
	case string:
		switch result.(string) {
		case "PARAM_ERROR":
			r = CHAT_MESSAGE_EVALSHA_PARAM_ERROR
		case "EXISTED":
			r = CHAT_MESSAGE_REPEATED_MESSAGE
		case "OK":
			ok = true
		default:
			r = CHAT_MESSAGE_REDIS_OPERATION_FAILED
			xlog.Warn(r)
		}
	default:
		r = CHAT_MESSAGE_REDIS_OPERATION_FAILED
		xlog.Warn(result)
	}
	return
}

func (c *chatMessageCache) IncrSeqID(chatId int64) (seqId int64, err error) {
	if seqId, err = xredis.IncrSeqID(chatId); err != nil {
		xlog.Warn(ERROR_CODE_CACHE_GET_SEQ_ID_FAILED, ERROR_CACHE_GET_SEQ_ID_FAILED, err.Error())
	}
	return
}

func (c *chatMessageCache) GetMaxSeqID(chatId int64) (seqId uint64, err error) {
	if seqId, err = xredis.GetMaxSeqID(chatId); err != nil {
		if err != redis.Nil {
			err = nil
		} else {
			xlog.Warn(ERROR_CODE_CACHE_REDIS_GET_FAILED, ERROR_CACHE_REDIS_GET_FAILED, err.Error())
		}
	}
	return
}

func (c *chatMessageCache) MGetMessages(keys ...string) ([]interface{}, error) {
	return xredis.MGet(keys...)
}
