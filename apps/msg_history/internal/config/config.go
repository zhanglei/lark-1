package config

import (
	"flag"
	"lark/pkg/common/xlog"
	"lark/pkg/conf"
	"lark/pkg/utils"
)

type Config struct {
	Name        string              `yaml:"name"`
	ServerID    int                 `yaml:"server_id"`
	Log         string              `yaml:"log"`
	Etcd        *conf.Etcd          `yaml:"etcd"`
	Mysql       *conf.Mysql         `yaml:"mysql"`
	Redis       *conf.Redis         `yaml:"redis"`
	MsgConsumer *conf.KafkaConsumer `yaml:"msg_consumer"`
}

var (
	config = new(Config)
)

var (
	confFile = flag.String("cfg", "./configs/msg_history.yaml", "config file")
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
