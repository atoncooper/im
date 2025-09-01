package utils

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/websocket"
)

func TestNewWorkPools(t *testing.T) {
	pools := NewWsPool(32)
	defer pools.CloseWsPools()

	// 创建ws连接对象
	var c *websocket.Conn

	// 1. 启动测试服务端
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		up := websocket.Upgrader{}
		conn, err := up.Upgrade(w, r, nil)
		if err != nil {
			t.Error(err)
			return
		}
		// 服务端不主动关闭，留给测试用
		c = conn
	}))
	defer srv.Close()

	// 2. 客户端主动连接，触发 Upgrade
	wsURL := "ws" + srv.URL[len("http"):]
	clientConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer clientConn.Close()

	// 3. 此时 c 已经被赋值为服务端侧的 *websocket.Conn
	pools.AddConnTemplate("lh", c)

	// 4. 取出验证
	conn, ok := pools.GetConnTemplate("lh")
	if !ok {
		t.Error("没有拿到conn")
	}
	fmt.Println("[INFO] conn is", conn)

	pools.ReleaseConnTemplate("lh")

	_, ok = pools.GetConnTemplate("lh")
	if !ok {
		t.Error("conn 已经被销毁")
	}
}
