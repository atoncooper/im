package core

import (
	"context"
	"encoding/json"
	"gateway/config"
	"gateway/dto"
	"gateway/service"
	"gateway/utils"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/gorilla/websocket"
)

// default ServerId, but we dont advice use this value
var SERVER_ID = config.GatewayCfg.Application.NodeId

var wsBufferPool = &sync.Pool{
	New: func() any { return make([]byte, 1024) },
}

var (
	pingInterval = 25 * time.Second
	pongWait     = 60 * time.Second
)

var upgrader = websocket.Upgrader{
	HandshakeTimeout:  5 * time.Second,
	ReadBufferSize:    1024 * 2,
	WriteBufferSize:   1024 * 2,
	WriteBufferPool:   wsBufferPool,
	EnableCompression: true,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	Error: func(w http.ResponseWriter, r *http.Request, status int, reason error) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		w.Write([]byte(`{"error":"` + reason.Error() + `"}`))
	},
}

func websocketServer(c *gin.Context) {

	// 鉴权uid
	uid := c.Query("uid")
	if uid == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "uid is required"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 初始化创建状态status
	utils.StatusTemplate().InitStatus(uid, utils.Meta{
		UserRemoteAddr: conn.RemoteAddr().Network(),
		Status:         "online",
		ServerId:       SERVER_ID,
		ServerAddr:     config.GatewayCfg.Application.Host,
	})

	defer func() {
		_ = conn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, "bye"),
			time.Now().Add(5*time.Second),
		)
		conn.Close()
		log.Printf("client %s disconnected", conn.RemoteAddr())
	}()

	cleanup := func() {
		// clear connection info

		err := utils.StatusTemplate().ClearStatus(uid)
		if err != nil {
			log.Printf("clear status failed: %v", err)
		}

		log.Default().Println("[INFO] 清除断开连接信息")
	}

	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	//
	pingTicker := time.NewTicker(pingInterval)
	defer pingTicker.Stop()

	defer func() {
		pingTicker.Stop()
		conn.WriteControl(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, "bye"), time.Now().Add(5*time.Second))
	}()

	// TODO : handle service logic
	done := make(chan struct{})
	go func() { // 发送消息端
		defer func() {
			cleanup()
			close(done)
		}()
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}

			// 处理收到的消息
			if err := handleMessage(msg); err != nil {
				conn.WriteMessage(websocket.TextMessage, []byte(err.Error()))
			}
			conn.WriteMessage(websocket.TextMessage, []byte("message received"))
		}
	}()

	go func() { // TODO : 接受消息队列消息队列并转发到前端
	}()

	for {
		select {
		case <-pingTicker.C:
			if err := conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(5*time.Second)); err != nil {
				return
			}
		default:
			time.Sleep(100 * time.Millisecond)
		}
	}
}

var validate = validator.New()

// handleMessage
//
// 处理消息函数
// 处理消息转发，若是处理失败则返回消息未能成功发送的结果，希望冲重新投入发送.
func handleMessage(msg []byte) error {
	var message *dto.MessageDTO

	if err := json.Unmarshal(msg, &message); err != nil {
		return err
	}
	if err := validate.Struct(message); err != nil {
		return err
	}

	// service hander
	data, err := utils.StatusTemplate().GetStatus(message.ReceiverId)
	if err != nil {
		return err
	}

	// 不同路由处理
	if data.ServerId != SERVER_ID && data.Status == "online" {
		// 消息队列发送处理
		err = service.NewProducer(
			config.KafkaProducerTemplate(),
			config.KafkaDLQTemplate(),
		).SendMessage(
			context.Background(),
			message,
			SERVER_ID,
		)
		if err != nil {
			return err
		}
	}

	// 同路由在线处理
	if data.ServerId == SERVER_ID && data.Status == "online" {
		// 直接发送
		conn, ok := utils.PoolsOpsTemplate().GetConnTemplate(data.UserRemoteAddr)
		if !ok {
			// TODO : 发送至service
		}
		err = conn.WriteMessage(websocket.TextMessage, msg)
		if err != nil {
			// TODO 发送至service
			return err
		}
	}

	if data.Status != "online" {
		// TODO : 发送至service
	}

	return nil
}
