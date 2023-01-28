package service

import (
	"lark/apps/server_mgr/internal/config"
	"lark/domain/cache"
	"lark/pkg/common/xetcd"
	"lark/pkg/common/xlog"
	"lark/pkg/utils"
)

type ServerMgrService interface {
	Run()
}

type serverMgrService struct {
	cfg            *config.Config
	watcher        *xetcd.Watcher
	serverMgrCache cache.ServerMgrCache
}

func NewServerMgrService(cfg *config.Config, serverMgrCache cache.ServerMgrCache) ServerMgrService {
	return &serverMgrService{cfg: cfg, serverMgrCache: serverMgrCache}
}

func (s *serverMgrService) Run() {
	s.watchMessageGateway()
}

func (s *serverMgrService) watchMessageGateway() {
	var (
		catalog = s.cfg.Etcd.Schema + ":///" + s.cfg.MsgGatewayServer.Name
		members = map[string]string{}
		servers []string
		srv     string
		ok      bool
		err     error
	)
	s.watcher, err = xetcd.NewWatcher(catalog, s.cfg.Etcd.Schema, s.cfg.Etcd.Endpoints, s.changeWatcher)
	if err != nil {
		xlog.Error(err.Error())
		return
	}
	err = s.watcher.Run()
	if err != nil {
		xlog.Error(err.Error())
		return
	}
	s.watcher.EachKvs(func(k string, v *xetcd.KeyValue) bool {
		var (
			name, port = utils.GetServer(k)
			portVal, _ = utils.ToInt(port)
			member     = name + ":" + utils.IntToStr(portVal+1)
		)
		members[member] = member
		return true
	})

	servers = s.serverMgrCache.ZRangeMsgGateway(0, -1)
	for _, srv = range servers {
		if _, ok = members[srv]; ok {
			continue
		}
		err = s.serverMgrCache.ZRemMsgGateway(srv)
		if err != nil {
			xlog.Warn(err.Error())
		}
	}

	s.watcher.Watch()
}

func (s *serverMgrService) changeWatcher(kv *xetcd.KeyValue, eventType int) {
	var (
		name, port = utils.GetServer(kv.Key)
		portVal, _ = utils.ToInt(port)
		member     = name + ":" + utils.IntToStr(portVal+1)
		err        error
	)
	switch eventType {
	case xetcd.EVENT_TYPE_PUT:
		err = s.serverMgrCache.ZAddMsgGateway(0, member)
		if err != nil {
			xlog.Warn(err.Error())
		}
	case xetcd.EVENT_TYPE_DELETE:
		err = s.serverMgrCache.ZRemMsgGateway(member)
		if err != nil {
			xlog.Warn(err.Error())
		}
	}
}
