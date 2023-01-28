package config

import (
	"flag"
	"lark/pkg/common/xlog"
	"lark/pkg/conf"
	"lark/pkg/utils"
)

type Config struct {
	Name             string              `yaml:"name"`
	ServerID         int                 `yaml:"server_id"`
	Log              string              `yaml:"log"`
	GrpcServer       *conf.Grpc          `yaml:"grpc_server"`
	ChatMemberServer *conf.GrpcServer    `yaml:"chat_member_server"`
	Etcd             *conf.Etcd          `yaml:"etcd"`
	Redis            *conf.Redis         `yaml:"redis"`
	MsgProducer      *conf.KafkaProducer `yaml:"msg_producer"`
	Jaeger           *conf.Jaeger        `yaml:"jaeger"`
}

var (
	config = new(Config)
)

var (
	confFile = flag.String("cfg", "./configs/message.yaml", "config file")
	serverId = flag.Int("sid", 1, "server id")
	grpcPort = flag.Int("gp", 7200, "grpc server port")
)

func init() {
	flag.Parse()
	utils.YamlToStruct(*confFile, config)

	config.ServerID = *serverId
	config.GrpcServer.ServerID = config.ServerID
	config.GrpcServer.Port = *grpcPort

	xlog.Shared(config.Log, config.Name+utils.IntToStr(config.ServerID))
}

func NewConfig() *Config {
	return config
}

func GetConfig() *Config {
	return config
}
