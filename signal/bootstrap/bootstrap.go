package bootstrap

import (
	"context"
	"fmt"
	"hash/crc32"
	"signal/configs"
	"signal/services"
	"time"

	pb "github.com/atoncooper/im/proto/seq"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
)

func init() {
	v := viper.New()

	v.SetConfigName("signal")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")

	v.SetDefault("server.port", 8080)
	v.SetDefault("server.read_timeout", "30s")

	v.SetEnvPrefix("APP")
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		panic(err)
	}

	var c configs.Config

	if err := v.Unmarshal(&c); err != nil {
		panic(err)
	}
	configs.Cfg.Store(&c)

	v.OnConfigChange(func(e fsnotify.Event) {
		var newC configs.Config
		if err := v.Unmarshal(&newC); err != nil {
			panic(err)
		}
		configs.Cfg.Store(&newC)
	})
	v.WatchConfig()

	// 初始化redis
	redisCfg := configs.RedisConfig{
		Addrs:    c.Application.Component.Redis.Nodes,
		Timeout:  10 * time.Second,
		Password: "",
		PoolSize: 10,
	}

	redisCil := configs.NewRedisClient(&redisCfg)
	configs.Client.Store(redisCil)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := redisCil.Ping(ctx).Err(); err != nil {
		panic(err)
	}

	// 注册consul服务

	client, err := configs.NewEnterpriseClient([]string{"127.0.0.1"}, "", "")
	if err != nil {
		panic(err)
	}
	consulClient := &configs.ConsulClient{Client: client}

	// TODO : 注册服务
	// 获取机器id
	nodeID, err := consulClient.GetNodeID()
	if err != nil {
		panic(err)
	}

	var machineID int64
	_, err = fmt.Sscanf(nodeID, "%x", &machineID)
	if err != nil {
		machineID = int64(crc32.ChecksumIEEE([]byte(nodeID))) % 1024
	}

	// 初始化生成器

	err = services.InitGenerator(machineID, redisCil)
	if err != nil {
		panic(err)
	}

	// 初始化grpc服务
	gRPCcfg := services.GRPCConfig{
		Host: c.Application.Server.Host,
		Port: c.Application.Server.Port,
	}

	// 注册register服务
	register := func(s *grpc.Server) {
		pb.RegisterSequenceServiceServer(s, &services.ServerHandle{
			Redis: configs.RedisTmeplate(),
		})
	}

	services.NewGRPCServer(&gRPCcfg, register)

}
