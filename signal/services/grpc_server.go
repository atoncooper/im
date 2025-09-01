package services

import (
	"fmt"
	"net"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

type GRPCConfig struct {
	Host string
	Port int

	MaxConnIdle      time.Duration
	MaxConnAge       time.Duration
	MaxConcurrent    uint32
	KeepAlive        bool
	KeepAliveTime    time.Duration
	KeepAliveTimeout time.Duration
}

func newDafaultgRPCCfg() *GRPCConfig {
	return &GRPCConfig{
		Host:             "127.0.0.1",
		Port:             50051,
		MaxConnIdle:      30 * time.Second,
		MaxConnAge:       2 * time.Minute,
		MaxConcurrent:    1000,
		KeepAlive:        true,
		KeepAliveTime:    30 * time.Second,
		KeepAliveTimeout: 10 * time.Second,
	}
}

var (
	srv     *grpc.Server
	srvOnce sync.Once
)

func NewGRPCServer(cfg *GRPCConfig, register func(*grpc.Server)) {
	srvOnce.Do(func() {
		if cfg == nil {
			cfg = newDafaultgRPCCfg()
		}

		kp := keepalive.ServerParameters{
			MaxConnectionIdle:     cfg.MaxConnIdle,
			MaxConnectionAge:      cfg.MaxConnAge,
			MaxConnectionAgeGrace: 5 * time.Second,
			Time:                  cfg.KeepAliveTime,
			Timeout:               cfg.KeepAliveTimeout,
		}

		srv = grpc.NewServer(
			grpc.KeepaliveParams(kp),
			grpc.MaxConcurrentStreams(cfg.MaxConcurrent),
		)

		register(srv)

		go func() {
			lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", cfg.Host, cfg.Port))
			if err != nil {
				panic(err)
			}
			_ = srv.Serve(lis)
		}()
	})

}

func GRPCTemplate() *grpc.Server {
	if srv == nil {
		panic("gRPC server not initialized; call services.Init first")
	}
	return srv
}

func GRPCstop() {
	if srv != nil {
		srv.GracefulStop()
	}
}
