package core

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
)

type GRpcConfig struct {
	ServerAddr       string
	Timeout          time.Duration
	TLS              bool
	TLSFile          string
	TLSKey           string
	TLSCac           string
	KeepAliveTime    time.Duration
	KeepAliveTimeout time.Duration
}

var (
	instance *grpc.ClientConn
	once     sync.Once
	mu       sync.RWMutex
)

func newDefaultgRPCConfig() *GRpcConfig {
	return &GRpcConfig{
		ServerAddr:       "localhost:50051",
		Timeout:          10 * time.Second,
		TLS:              false,
		TLSFile:          "",
		TLSKey:           "",
		TLSCac:           "",
		KeepAliveTime:    10 * time.Second,
		KeepAliveTimeout: 5 * time.Second,
	}
}

type GRpcParams struct {
	defineParams  *GRpcConfig
	defaultParams string
}
type GRpcParamsOps func(*GRpcParams)

func DefineGRPCParams(p *GRpcConfig) GRpcParamsOps {
	return func(params *GRpcParams) {
		params.defineParams = p
	}
}
func DefaultGRPCParams(p string) GRpcParamsOps {
	return func(params *GRpcParams) {
		params.defaultParams = p
	}
}

func loadTLS(cfg *GRpcConfig) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(cfg.TLSFile, cfg.TLSKey)
	if err != nil {
		return nil, err
	}
	// load ca certificate
	caCert, err := os.ReadFile(cfg.TLSCac)
	if err != nil {
		return nil, err
	}
	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to append CA certificate")
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
	}, nil
}

func gRPCClientServer(grpcOpts ...GRpcParamsOps) error {

	params := &GRpcParams{}
	for _, opt := range grpcOpts {
		opt(params)
	}
	if params.defineParams == nil {
		params.defineParams = newDefaultgRPCConfig()
	}

	cfg := params.defineParams

	opts := []grpc.DialOption{
		grpc.WithTimeout(cfg.Timeout),
		grpc.WithBlock(),
	}

	// check tls config
	if cfg.TLS {
		tlsCfg, error := loadTLS(cfg)
		if error != nil {
			return error
		}
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(tlsCfg)))
	}
	keepLivecfg := keepalive.ClientParameters{
		Time:                cfg.KeepAliveTime,
		Timeout:             cfg.KeepAliveTimeout,
		PermitWithoutStream: true,
	}
	opts = append(opts, grpc.WithKeepaliveParams(keepLivecfg))
	conn, err := grpc.NewClient(cfg.ServerAddr, opts...)
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	instance = conn
	log.Default().Println("[INFO] gRPC client connected to server")
	return nil
}

func NewgRPCServer(grpcOpts ...GRpcParamsOps) {
	var err error
	once.Do(func() {
		err = gRPCClientServer(grpcOpts...)
		if err != nil {
			panic(err)
		}
	})
}

func GetgRPCConn() (*grpc.ClientConn, error) {
	mu.RLock()
	defer mu.RUnlock()
	if instance == nil {
		return nil, fmt.Errorf("gRPC client not connected")
	}
	return instance, nil
}

func ReleasegRPCConn() error {
	mu.Lock()
	defer mu.Unlock()
	if instance != nil {
		instance.Close()
		instance = nil
		log.Default().Println("[INFO] gRPC client connection released")
	}
	return nil
}
