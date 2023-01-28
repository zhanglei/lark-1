package service

import (
	"context"
	chat_client "lark/apps/chat/client"
	"lark/apps/chat_invite/internal/config"
	dist_client "lark/apps/dist/client"
	user_client "lark/apps/user/client"
	"lark/domain/cache"
	"lark/domain/repo"
	"lark/pkg/common/xkafka"
	"lark/pkg/proto/pb_invite"
)

type ChatInviteService interface {
	InitiateChatInvite(ctx context.Context, req *pb_invite.InitiateChatInviteReq) (resp *pb_invite.InitiateChatInviteResp, err error)
	ChatInviteHandle(ctx context.Context, req *pb_invite.ChatInviteHandleReq) (resp *pb_invite.ChatInviteHandleResp, err error)
	ChatInviteList(ctx context.Context, req *pb_invite.ChatInviteListReq) (resp *pb_invite.ChatInviteListResp, err error)
}

type chatInviteService struct {
	conf             *config.Config
	chatInviteRepo   repo.ChatInviteRepository
	userRepo         repo.UserRepository
	avatarRepo       repo.AvatarRepository
	chatMemberRepo   repo.ChatMemberRepository
	chatRepo         repo.ChatRepository
	chatCache        cache.ChatCache
	chatMessageCache cache.ChatMessageCache
	chatMemberCache  cache.ChatMemberCache
	userCache        cache.UserCache
	chatClient       chat_client.ChatClient
	userClient       user_client.UserClient
	distClient       dist_client.DistClient
	producer         *xkafka.Producer
}

func NewChatInviteService(
	conf *config.Config,
	chatInviteRepo repo.ChatInviteRepository,
	userRepo repo.UserRepository,
	avatarRepo repo.AvatarRepository,
	chatMemberRepo repo.ChatMemberRepository,
	chatRepo repo.ChatRepository,
	chatCache cache.ChatCache,
	chatMessageCache cache.ChatMessageCache,
	chatMemberCache cache.ChatMemberCache,
	userCache cache.UserCache,
) ChatInviteService {
	svc := &chatInviteService{
		conf:             conf,
		chatInviteRepo:   chatInviteRepo,
		userRepo:         userRepo,
		avatarRepo:       avatarRepo,
		chatMemberRepo:   chatMemberRepo,
		chatRepo:         chatRepo,
		chatCache:        chatCache,
		chatMessageCache: chatMessageCache,
		chatMemberCache:  chatMemberCache,
		userCache:        userCache}
	svc.chatClient = chat_client.NewChatClient(conf.Etcd, conf.ChatServer, conf.Jaeger, conf.Name)
	svc.userClient = user_client.NewUserClient(conf.Etcd, conf.UserServer, conf.Jaeger, conf.Name)
	svc.distClient = dist_client.NewDistClient(conf.Etcd, conf.DistServer, conf.Jaeger, conf.Name)
	svc.producer = xkafka.NewKafkaProducer(conf.MsgProducer.Address, conf.MsgProducer.Topic)
	return svc
}
