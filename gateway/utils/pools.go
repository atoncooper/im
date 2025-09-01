package utils

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"

	"github.com/gorilla/websocket"
)

type Pools struct {
	conns  sync.Map
	ctx    context.Context
	cancel context.CancelFunc
	max    uint32
	cnt    uint32 // 当前连接数
}

type WsConn struct {
	*websocket.Conn
	clientID string
	pool     *Pools
}

func NewWsPool(maxConn uint32) *Pools {
	ctx, cancel := context.WithCancel(context.Background())
	return &Pools{
		ctx:    ctx,
		cancel: cancel,
		max:    maxConn,
	}
}

var (
	poolIns  *Pools
	poolOnce sync.Once
)

func InitWsPool() {
	poolOnce.Do(func() {
		poolIns = NewWsPool(400)
	})
}

func (p *Pools) AddConnTemplate(clientId string, conn *websocket.Conn) error {
	if atomic.LoadUint32(&p.cnt) >= p.max {
		return errors.New("connection pool full")
	}
	c := &WsConn{
		Conn:     conn,
		clientID: clientId,
		pool:     p,
	}
	p.conns.Store(clientId, c)
	atomic.AddUint32(&p.cnt, 1)
	return nil
}

func (p *Pools) ReleaseConnTemplate(clientId string) {
	if _, ok := p.conns.LoadAndDelete(clientId); ok {
		atomic.AddUint32(&p.cnt, ^uint32(0))
	}
}

func (p *Pools) GetConnTemplate(clientId string) (*websocket.Conn, bool) {
	v, ok := p.conns.Load(clientId)
	if !ok {
		return nil, false
	}
	wsConn := v.(*WsConn).Conn
	return wsConn, true
}

func (p *Pools) CloseWsPools() {
	p.cancel()
	p.conns.Range(func(_, v any) bool {
		conn := v.(*WsConn)
		_ = conn.Close()
		return true
	})
	p.conns = sync.Map{}
	atomic.StoreUint32(&p.cnt, 0)
}

func PoolsOpsTemplate() *Pools {
	if poolIns == nil {
		panic("pools: call InitWsPool first")
	}
	return poolIns
}
