package core

import (
	"context"
	"fmt"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type GrpcConfig struct {
	Host         string
	Port         int
	TLS          *TLSConfig
	Network      string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	StopTimeout  time.Duration
}

type TLSConfig struct {
	CertFile string
	KeyFile  string
}

func newDefaultGrpcConfig() *GrpcConfig {
	return &GrpcConfig{
		Host:         "0.0.0.0",
		Port:         50051,
		Network:      "tcp",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		StopTimeout:  5 * time.Second,
	}
}

func StartgRPCServer(cfg *GrpcConfig) {
	if cfg == nil {
		cfg = newDefaultGrpcConfig()
	}

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	lis, err := net.Listen(cfg.Network, addr)
	if err != nil {
		panic(err)
	}

	var opts []grpc.ServerOption
	if cfg.TLS != nil {
		creds, err := credentials.NewServerTLSFromFile(cfg.TLS.CertFile, cfg.TLS.KeyFile)
		if err != nil {
			panic(err)
		}
		opts = append(opts, grpc.Creds(creds))
	}

	// start grpc server
	srv := grpc.NewServer(opts...)

	// TODO: register your gRPC services here
	// pb.RegisterXXXServer(srv, &xxxServer{})

	go func() {
		if err := srv.Serve(lis); err != nil {
			panic(err)
		}
	}()

	<-context.Background().Done()
	_, cancel := context.WithTimeout(context.Background(), cfg.StopTimeout)
	defer cancel()
	srv.GracefulStop()
}
