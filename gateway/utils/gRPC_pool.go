package utils

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
)

// grpc 客户端配置项
type GRPCClientConfig struct {
	MaxConn             int           // 最大连接数
	Timeout             time.Duration // 连接超时时间
	Addr                string        // 服务端地址
	ServerName          string        // 服务器名称
	TLS                 bool          // 是否启用TLS加密
	TlsCertFile         string        // TLS证书文件路径
	TlsKeyFile          string        // TLS密钥文件路径
	TlsCACertFile       string        // CA证书文件路径
	KeepAliveTime       time.Duration // 客户端发送ping消息的时间间隔
	KeepAliveTimeout    time.Duration // 等待ping响应的超时时间
	PermitWithoutStream bool          // 是否允许在没有活跃流的情况下发送keepalive ping
	IdleTimeout         time.Duration // 空闲连接的超时时间
	CreatedAt           time.Time     // 连接创建时间
}

// gRPC连接包装器，包含实际连接和使用状态
// 用于连接池内部管理连接的生命周期和使用状态

type grpcConnection struct {
	conn     *grpc.ClientConn  // 实际的gRPC连接
	inUse    bool              // 标记连接是否正在使用中
	lastUsed time.Time         // 上次使用时间，用于空闲检测
	config   *GRPCClientConfig // 连接配置
	mu       sync.Mutex        // 保护连接状态的互斥锁
}

// gRPC 客户端连接池
// 支持多服务地址的连接管理，自动维护连接生命周期

type grpcClientPool struct {
	// 存储不同服务地址对应的连接列表
	connections map[string][]*grpcConnection
	// 连接配置
	config *GRPCClientConfig

	// 保护连接池的互斥锁
	mu sync.RWMutex
	// 用于清理空闲连接的定时器
	cleanupTicker *time.Ticker
	// 关闭通道，用于停止清理定时器
	stopChan chan struct{}
}

// NewgRPCWorkPool 创建一个新的gRPC客户端连接池
// 参数：配置信息
// 返回：初始化好的连接池指针

func NewgRPCWorkPool(config *GRPCClientConfig) (*grpcClientPool, error) {
	// 验证配置参数
	if config == nil {
		return nil, errors.New("grpc client config cannot be nil")
	}

	// 确保必要的配置项有效
	if config.MaxConn <= 0 {
		config.MaxConn = 10 // 默认最大连接数
	}
	if config.Timeout <= 0 {
		config.Timeout = 10 * time.Second // 默认连接超时时间
	}
	if config.Addr == "" {
		return nil, errors.New("server address cannot be empty")
	}
	if config.KeepAliveTime <= 0 {
		config.KeepAliveTime = 30 * time.Second // 默认keepalive时间间隔
	}
	if config.KeepAliveTimeout <= 0 {
		config.KeepAliveTimeout = 5 * time.Second // 默认keepalive超时时间
	}
	if config.IdleTimeout <= 0 {
		config.IdleTimeout = 5 * time.Minute // 默认空闲超时时间
	}

	// 创建连接池实例
	pool := &grpcClientPool{
		connections:   make(map[string][]*grpcConnection),
		config:        config,
		cleanupTicker: time.NewTicker(30 * time.Second), // 每30秒清理一次空闲连接
		stopChan:      make(chan struct{}),
	}

	// 启动空闲连接清理goroutine
	go pool.cleanupIdleConnections()

	return pool, nil
}

// loadTLS 加载TLS配置
// 参数：配置信息
// 返回：TLS配置和可能的错误

func (p *grpcClientPool) loadTLS(cfg *GRPCClientConfig) (*tls.Config, error) {
	if !cfg.TLS {
		return nil, nil
	}

	// 加载客户端证书和密钥
	cert, err := tls.LoadX509KeyPair(cfg.TlsCACertFile, cfg.TlsKeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load client certificate: %v", err)
	}

	// 加载CA证书
	caCert, err := os.ReadFile(cfg.TlsCACertFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load CA certificate: %v", err)
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return nil, errors.New("failed to append CA certificate")
	}

	// 创建并返回TLS配置
	return &tls.Config{
			Certificates: []tls.Certificate{cert},
			RootCAs:      caCertPool,
		},
		nil
}

// createConnection 创建一个新的gRPC连接
// 参数：服务地址
// 返回：连接包装器和可能的错误

func (p *grpcClientPool) createConnection(addr string) (*grpcConnection, error) {
	// 创建gRPC连接选项
	opts := []grpc.DialOption{
		grpc.WithTimeout(p.config.Timeout),
		grpc.WithBlock(),
	}

	// 如果配置了TLS，则加载TLS配置
	if p.config.TLS {
		tlsCfg, err := p.loadTLS(p.config)
		if err != nil {
			return nil, err
		}
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(tlsCfg)))
	} else {
		// 不使用TLS时，使用不安全的连接
		opts = append(opts, grpc.WithInsecure())
	}

	// 配置keepalive参数
	keepAliveCfg := keepalive.ClientParameters{
		Time:                p.config.KeepAliveTime,
		Timeout:             p.config.KeepAliveTimeout,
		PermitWithoutStream: p.config.PermitWithoutStream,
	}
	opts = append(opts, grpc.WithKeepaliveParams(keepAliveCfg))

	// 创建实际的gRPC连接
	conn, err := grpc.Dial(addr, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %v", addr, err)
	}

	// 创建连接包装器
	connection := &grpcConnection{
		conn:     conn,
		inUse:    false,
		lastUsed: time.Now(),
		config:   p.config,
	}

	return connection, nil
}

// GetConnection 从连接池中获取一个可用的连接
// 参数：服务地址
// 返回：gRPC连接和可能的错误

func (p *grpcClientPool) GetConnection(addr string) (*grpc.ClientConn, error) {
	p.mu.RLock()
	connections, exists := p.connections[addr]
	p.mu.RUnlock()

	// 检查是否有可用连接
	if exists {
		for _, conn := range connections {
			conn.mu.Lock()
			if !conn.inUse {
				conn.inUse = true
				conn.lastUsed = time.Now()
				conn.mu.Unlock()
				return conn.conn, nil
			}
			conn.mu.Unlock()
		}

		// 如果所有连接都在使用中，但未达到最大连接数，则创建新连接
		p.mu.RLock()
		if len(p.connections[addr]) < p.config.MaxConn {
			p.mu.RUnlock()
			newConn, err := p.createConnection(addr)
			if err != nil {
				return nil, err
			}

			newConn.inUse = true
			newConn.lastUsed = time.Now()

			p.mu.Lock()
			p.connections[addr] = append(p.connections[addr], newConn)
			p.mu.Unlock()

			return newConn.conn, nil
		}
		p.mu.RUnlock()
	} else {
		// 第一次请求该服务地址，创建连接
		newConn, err := p.createConnection(addr)
		if err != nil {
			return nil, err
		}

		newConn.inUse = true
		newConn.lastUsed = time.Now()

		p.mu.Lock()
		p.connections[addr] = []*grpcConnection{newConn}
		p.mu.Unlock()

		return newConn.conn, nil
	}

	// 所有连接都在使用中且达到最大连接数
	return nil, errors.New("all connections are in use and max connection limit reached")
}

// ReleaseConnection 释放一个连接回连接池
// 参数：服务地址和要释放的连接
// 返回：可能的错误

func (p *grpcClientPool) ReleaseConnection(addr string, conn *grpc.ClientConn) error {
	p.mu.RLock()
	connections, exists := p.connections[addr]
	p.mu.RUnlock()

	if !exists {
		return errors.New("connection not found in pool")
	}

	// 查找对应的连接包装器
	for _, c := range connections {
		c.mu.Lock()
		if c.conn == conn {
			c.inUse = false
			c.lastUsed = time.Now()
			c.mu.Unlock()
			return nil
		}
		c.mu.Unlock()
	}

	return errors.New("connection not found in pool")
}

// Close 关闭连接池中的所有连接
// 返回：可能的错误

func (p *grpcClientPool) Close() error {
	// 停止清理定时器
	close(p.stopChan)
	p.cleanupTicker.Stop()

	p.mu.Lock()
	defer p.mu.Unlock()

	var lastErr error

	// 关闭所有连接
	for addr, connections := range p.connections {
		for _, conn := range connections {
			if conn.conn != nil {
				err := conn.conn.Close()
				if err != nil {
					log.Printf("Error closing connection to %s: %v", addr, err)
					lastErr = err
				}
			}
		}
	}

	// 清空连接池
	p.connections = make(map[string][]*grpcConnection)

	return lastErr
}

// cleanupIdleConnections 定期清理空闲连接
// 防止资源泄漏，自动关闭长时间未使用的连接

func (p *grpcClientPool) cleanupIdleConnections() {
	for {
		select {
		case <-p.cleanupTicker.C:
			p.mu.Lock()
			for addr, connections := range p.connections {
				var activeConnections []*grpcConnection
				for _, conn := range connections {
					conn.mu.Lock()
					// 检查连接是否空闲超时且未在使用中
					if !conn.inUse && time.Since(conn.lastUsed) > p.config.IdleTimeout {
						// 关闭空闲超时的连接
						if conn.conn != nil {
							conn.conn.Close()
						}
						log.Printf("Closed idle connection to %s", addr)
					} else {
						// 保留活跃连接
						activeConnections = append(activeConnections, conn)
					}
					conn.mu.Unlock()
				}
				// 更新连接列表，只保留活跃连接
				p.connections[addr] = activeConnections
				// 如果该服务地址没有活跃连接，从map中删除
				if len(activeConnections) == 0 {
					delete(p.connections, addr)
				}
			}
			p.mu.Unlock()
		case <-p.stopChan:
			// 收到停止信号，退出清理goroutine
			return
		}
	}
}

// GetConnectionCount 获取指定服务地址的连接数量
// 参数：服务地址
// 返回：连接数量

func (p *grpcClientPool) GetConnectionCount(addr string) int {
	p.mu.RLock()
	defer p.mu.RUnlock()

	connections, exists := p.connections[addr]
	if !exists {
		return 0
	}

	return len(connections)
}

// 单例模式的连接池实例
var (
	poolInstance *grpcClientPool
	rpcPoolOnce  sync.Once
)

// GetgRPCPoolInstance 获取连接池的单例实例
// 参数：配置信息（如果是第一次调用）
// 返回：连接池实例和可能的错误

func GetgRPCPoolInstance(config *GRPCClientConfig) (*grpcClientPool, error) {
	var err error
	rpcPoolOnce.Do(func() {
		poolInstance, err = NewgRPCWorkPool(config)
	})

	// 如果已经初始化过，但传入了配置，更新配置
	// 注意：这里只更新全局配置，已创建的连接不会受影响
	if err == nil && config != nil {
		if poolInstance != nil {
			poolInstance.mu.Lock()
			poolInstance.config = config
			poolInstance.mu.Unlock()
		}
	}

	return poolInstance, err
}
