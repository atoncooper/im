package config

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisConf struct {
	Addrs       []string
	Password    string
	PoolMaxIdle int
	MaxActive   int
	IdelTimeout time.Duration
}

var (
	rc   *redis.ClusterClient
	once sync.Once
)

func NewRedisCluster(cfg *RedisConf) (*redis.ClusterClient, error) {
	var err error
	once.Do(func() {
		opts := &redis.ClusterOptions{
			Addrs:          cfg.Addrs,
			Password:       cfg.Password,
			PoolSize:       cfg.PoolMaxIdle,
			MaxActiveConns: cfg.MaxActive,
			DialTimeout:    cfg.IdelTimeout,
		}

		rc = redis.NewClusterClient(opts)
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if _, e := rc.Ping(ctx).Result(); e != nil {
			err = fmt.Errorf("redis ping fail: %w", e)
			rc = nil
		}
	})
	if rc == nil && err == nil {
		err = fmt.Errorf("redis not initialized")
	}
	return rc, err
}

func RedisTemplate() *redis.ClusterClient {
	return rc
}

func RedisClose(ctx context.Context) error {
	if rc == nil {
		return nil
	}
	return rc.Close()
}
