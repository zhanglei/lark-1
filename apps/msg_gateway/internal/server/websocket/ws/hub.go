package ws

import (
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"
	"lark/pkg/proto/pb_msg"
	"lark/pkg/utils"
	"log"
	"runtime/debug"
	"strconv"
	"sync/atomic"
	"time"
)

type Hub struct {
	serverId         int
	upgrader         websocket.Upgrader
	readChan         chan *Message   // 客户端发送的消息
	msgCallback      MessageCallback // 回调
	clients          *CliMap         // key:platform-uid
	sentMessageCount int64
}

func NewHub(serverId int, msgCallback MessageCallback) *Hub {
	return &Hub{
		serverId: serverId,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  WS_READ_BUFFER_SIZE,
			WriteBufferSize: WS_WRITE_BUFFER_SIZE,
		},
		readChan:    make(chan *Message, WS_CHAN_SERVER_READ_MESSAGE_SIZE),
		msgCallback: msgCallback,
		clients:     NewCliMap(),
	}
}

func (h *Hub) registerClient(client *Client) {
	var (
		ok  bool
		cli *Client
	)

	if cli, ok = h.clients.Get(client.key); ok == false {
		h.clients.Set(client.key, client)
		h.OnOffline(true)
		return
	}

	if client.onlineTs > cli.onlineTs {
		h.clients.Set(client.key, client)
		h.close(cli)
		return
	}
	h.close(client)
}

func (h *Hub) close(client *Client) {
	client.Close()
}

func (h *Hub) unregisterClient(client *Client) {
	var (
		ok  bool
		cli *Client
	)
	if cli, ok = h.clients.Get(client.key); ok == false {
		return
	}
	if client == cli {
		h.clients.Delete(client.key)
		h.OnOffline(false)
	}
}

func (h *Hub) OnOffline(onOff bool) {
	if onOff {
		log.Println("R在线用户数量:", h.clients.Len())
	} else {
		log.Println("U在线用户数量:", h.clients.Len())
	}
}

func (h *Hub) Run() {
	defer func() {
		if r := recover(); r != nil {
			wsLog.Error(r, string(debug.Stack()))
		}
	}()

	// 调试用
	h.debug()

	var (
		index int
		loop  = WS_MAX_CONSUMER_SIZE
	)
	for index = 0; index < loop; index++ {
		go func() {
			for {
				select {
				case msg := <-h.readChan:
					h.msgCallback(msg)
				}
			}
		}()
	}
}

func (h *Hub) IsOnline(uid int64, platform int32) (ok bool) {
	_, ok = h.clients.Get(clientKey(uid, platform))
	return
}

func (h *Hub) SendMessage(uid int64, platform int32, message []byte) (result int32) {
	result = WS_CLIENT_OFFLINE
	var (
		cli *Client
		ok  bool
	)
	if cli, ok = h.clients.Get(clientKey(uid, platform)); ok == false {
		return
	}
	atomic.AddInt64(&h.sentMessageCount, 1)
	cli.Send(message)
	result = WS_SEND_MSG_SUCCESS
	return
}

func (h *Hub) NumberOfOnline() int64 {
	return int64(h.clients.Len())
}

func (h *Hub) broadcastMessage() {
	var (
		msgBuf, _       = proto.Marshal(&pb_msg.SrvChatMessage{})
		broadcastBuf, _ = utils.Encode(1, 0, 0, msgBuf)
		uid             int64
	)
	for uid = 1; uid <= 10000; uid++ {
		h.SendMessage(uid, 1, broadcastBuf)
	}
}

func (h *Hub) debug() {
	go func() {
		allTicker := time.NewTicker(time.Second * 60)
		defer allTicker.Stop()
		for {
			select {
			case <-allTicker.C:
				log.Println("在线人数:", h.clients.Len(), " 发送消息数量:", h.sentMessageCount)
			}
		}
	}()
}

func (h *Hub) wsHandler(c *gin.Context) {
	var (
		uidVal   interface{}
		pidVal   interface{}
		exists   bool
		uid      int64
		platform int32
		conn     *websocket.Conn
		client   *Client
		err      error
	)

	if h.clients.Len() >= WS_MAX_CONNECTIONS {
		httpError(c, ERROR_CODE_WS_EXCEED_MAX_CONNECTIONS, ERROR_WS_EXCEED_MAX_CONNECTIONS)
		return
	}
	uidVal, exists = c.Get(WS_KEY_UID)
	if exists == false {
		httpError(c, ERROR_CODE_HTTP_UID_DOESNOT_EXIST, ERROR_HTTP_UID_DOESNOT_EXIST)
		return
	}
	pidVal, exists = c.Get(WS_KEY_PLATFORM)
	if exists == false {
		httpError(c, ERROR_CODE_HTTP_PLATFORM_DOESNOT_EXIST, ERROR_HTTP_PLATFORM_DOESNOT_EXIST)
		return
	}
	uid, _ = strconv.ParseInt(uidVal.(string), 10, 64)
	if uid == 0 {
		httpError(c, ERROR_CODE_HTTP_UID_DOESNOT_EXIST, ERROR_HTTP_UID_DOESNOT_EXIST)
		return
	}
	platform = int32(pidVal.(float64))

	if conn, err = h.upgrader.Upgrade(c.Writer, c.Request, nil); err != nil {
		// 协议升级失败
		httpError(c, ERROR_CODE_HTTP_UPGRADER_FAILED, err.Error())
		wsLog.Warn(err.Error())
		return
	}
	client = newClient(h, conn, uid, platform)
	client.listen()
	h.registerClient(client)
}
