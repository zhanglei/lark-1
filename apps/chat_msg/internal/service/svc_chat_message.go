package service

import (
	"context"
	chat_member_client "lark/apps/chat_member/client"
	"lark/apps/chat_msg/internal/config"
	"lark/domain/cache"
	"lark/domain/mrepo"
	"lark/domain/repo"
	"lark/pkg/common/xkafka"
	"lark/pkg/proto/pb_chat_msg"
)

type ChatMessageService interface {
	GetChatMessageList(ctx context.Context, req *pb_chat_msg.GetChatMessageListReq) (resp *pb_chat_msg.GetChatMessageListResp, err error)
	// 弃用
	GetChatMessages(ctx context.Context, req *pb_chat_msg.GetChatMessagesReq) (resp *pb_chat_msg.GetChatMessagesResp, err error)
	SearchMessage(ctx context.Context, req *pb_chat_msg.SearchMessageReq) (resp *pb_chat_msg.SearchMessageResp, err error)
	MessageOperation(ctx context.Context, req *pb_chat_msg.MessageOperationReq) (resp *pb_chat_msg.MessageOperationResp, err error)
}

type chatMessageService struct {
	conf             *config.Config
	chatMessageRepo  repo.ChatMessageRepository
	messageHotRepo   mrepo.MessageHotRepository
	chatMessageCache cache.ChatMessageCache
	chatMemberCache  cache.ChatMemberCache
	chatMemberClient chat_member_client.ChatMemberClient
	producer         *xkafka.Producer
}

func NewChatMessageService(
	conf *config.Config,
	chatMessageRepo repo.ChatMessageRepository,
	messageHotRepo mrepo.MessageHotRepository,
	chatMessageCache cache.ChatMessageCache,
	chatMemberCache cache.ChatMemberCache) ChatMessageService {
	svc := &chatMessageService{
		conf:             conf,
		chatMessageRepo:  chatMessageRepo,
		messageHotRepo:   messageHotRepo,
		chatMessageCache: chatMessageCache,
		chatMemberCache:  chatMemberCache}
	svc.chatMemberClient = chat_member_client.NewChatMemberClient(conf.Etcd, conf.ChatMemberServer, conf.GrpcServer.Jaeger, conf.Name)
	svc.producer = xkafka.NewKafkaProducer(conf.MsgProducer.Address, conf.MsgProducer.Topic)
	return svc
}
