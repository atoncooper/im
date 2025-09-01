package configs

import (
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisConfig struct {
	Addrs        []string
	Timeout      time.Duration
	Password     string
	PoolSize     int
	MinIdle      int
	MaxConnAge   time.Duration
	IdleTimeout  time.Duration
	PoolTimeout  time.Duration
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	MaxRetries   int
	ReadOnly     bool
	KeyPrefix    string
	// TLS
}

func newdefaultRedisConfig() *RedisConfig {
	return &RedisConfig{
		Addrs:        []string{"127.0.0.1:6379"},
		Password:     "",
		PoolSize:     200,
		MinIdle:      20,
		MaxConnAge:   30 * time.Minute,
		IdleTimeout:  5 * time.Minute,
		PoolTimeout:  2 * time.Second,
		DialTimeout:  3 * time.Second,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
		MaxRetries:   3,
		// TLSConfig:    nil,
		ReadOnly:  false,
		KeyPrefix: "im:",
	}
}

func NewRedisClient(cfg *RedisConfig) *redis.ClusterClient {
	if cfg == nil {
		cfg = newdefaultRedisConfig()
	}
	return redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:           cfg.Addrs,
		Password:        cfg.Password,
		PoolSize:        cfg.PoolSize,
		MinIdleConns:    cfg.MinIdle,
		ConnMaxLifetime: cfg.MaxConnAge,
		// IdleTimeout:  cfg.IdleTimeout,
		PoolTimeout:  cfg.PoolTimeout,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		MaxRetries:   cfg.MaxRetries,
		ReadOnly:     cfg.ReadOnly,
		// TLSConfig:    cfg.TLSConfig,
		// KeyPrefix: cfg.KeyPrefix,

	})
}

var (
	Client atomic.Pointer[redis.ClusterClient]
	once   sync.Once
)

func RedisTmeplate() *redis.ClusterClient {
	return Client.Load()
}

func StopRedis() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	<-c

	if cli := RedisTmeplate(); cli != nil {
		_ = cli.Close()
	}
}
