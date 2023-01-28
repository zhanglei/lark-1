package config

import (
	"flag"
	"lark/pkg/common/xlog"
	"lark/pkg/conf"
	"lark/pkg/utils"
)

type Config struct {
	Name             string           `yaml:"name"`
	ServerID         int              `yaml:"server_id"`
	Port             int              `yaml:"port"`
	Log              string           `yaml:"log"`
	Etcd             *conf.Etcd       `yaml:"etcd"`
	Redis            *conf.Redis      `yaml:"redis"`
	AuthServer       *conf.GrpcServer `yaml:"auth_server"`
	UserServer       *conf.GrpcServer `yaml:"user_server"`
	ChatMsgServer    *conf.GrpcServer `yaml:"chat_msg_server"`
	MessageServer    *conf.GrpcServer `yaml:"message_server"`
	LinkServer       *conf.GrpcServer `yaml:"link_server"`
	ChatMemberServer *conf.GrpcServer `yaml:"chat_member_server"`
	ChatInviteServer *conf.GrpcServer `yaml:"chat_invite_server"`
	ChatServer       *conf.GrpcServer `yaml:"chat_server"`
	AvatarServer     *conf.GrpcServer `yaml:"avatar_server"`
	ConvoServer      *conf.GrpcServer `yaml:"convo_server"`
	Minio            *conf.Minio      `yaml:"minio"`
	Jaeger           *conf.Jaeger     `yaml:"jaeger"`
}

var (
	config = new(Config)
)

var (
	confFile = flag.String("cfg", "./configs/api_gateway.yaml", "config file")
	serverId = flag.Int("sid", 1, "server id")
)

func init() {
	flag.Parse()
	utils.YamlToStruct(*confFile, config)

	config.ServerID = *serverId

	xlog.Shared(config.Log, config.Name+utils.IntToStr(config.ServerID))
}

func NewConfig() *Config {
	return config
}

func GetConfig() *Config {
	return config
}
