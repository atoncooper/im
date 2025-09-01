package main

import (
	"context"
	"fmt"
	"gateway/config"
	"gateway/core"
	"gateway/service"
	"gateway/utils"
	"log"
	"strings"
	"time"
)

func main() {

	// 初始化kafka
	cfg := config.GatewayCfg

	dlqTopic := cfg.Application.Component.Kafka.Topic + cfg.Application.Component.Kafka.Dlq.TopicSuffix

	kafkaConf := config.KafkaConf{
		Brokers:      []string{cfg.Application.Component.Kafka.Brokers},
		Topic:        cfg.Application.Component.Kafka.Topic,
		GroupId:      cfg.Application.Component.Kafka.GroupID,
		RequiredAcks: cfg.Application.Component.Kafka.RequiredAcks,
		DLQTopic:     dlqTopic,
		MaxRetry:     cfg.Application.Component.Kafka.MaxRetries,
	}

	config.NewKafak(&kafkaConf)

	// 初始化redis集群
	// 修复Redis连接地址解析问题：将逗号分隔的地址字符串拆分为多个地址元素
	redisNodes := strings.Split(cfg.Application.Component.Redis.Nodes, ",")
	redisConf := &config.RedisConf{
		Addrs:       redisNodes,
		Password:    cfg.Application.Component.Redis.Password,
		PoolMaxIdle: cfg.Application.Component.Redis.PoolSize,
		MaxActive:   cfg.Application.Component.Redis.MaxActive,
		IdelTimeout: 30 * time.Second,
	}

	rc, err := config.NewRedisCluster(redisConf)
	if err != nil {
		panic(err)
	}

	// 初始化consul配置

	addr := fmt.Sprintf("%s:%d", cfg.Application.Component.Consul.Endpoint, cfg.Application.Component.Consul.Port)

	consulConf := config.ConsulConf{
		Address: addr,
		Scheme:  cfg.Application.Component.Consul.Scheme,
	}

	config.NewConsul(&consulConf, config.ServerMeta{
		ID:      cfg.Application.NodeId,
		Name:    cfg.Application.Name,
		Host:    cfg.Application.Host,
		Port:    cfg.Application.Port,
		Tag:     cfg.Application.Tag,
		Check:   "http",
		Timeout: time.Duration(10) * time.Second,
	})

	// 初始化status
	utils.InitStatus(rc)

	// 初始化工作池
	utils.NewWsPool(1000)

	// 启动协程消费kafka
	go func() {
		receiver := service.NewReceviceMessage(
			config.KafkaConsumerTemplate(),
			config.KafkaDLQTemplate(),
		)
		err := receiver.StartReadMessage(context.Background(), cfg.Application.NodeId)
		if err != nil {
			log.Default().Printf("[ERROR] 消费消息失败: %v", err)
		}
		receiver.Wait()
	}()

	core.StartHttpServer(core.DefaultParams(""))
}
