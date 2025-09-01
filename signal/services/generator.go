package services

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/redis/go-redis/v9"
)

type IDGenerator struct {
	snowflakeNode *snowflake.Node
	redisClient   *redis.ClusterClient
	redisTimeout  time.Duration
}

func NewIDGenerator(machineId int64, redis *redis.ClusterClient) (*IDGenerator, error) {
	node, err := snowflake.NewNode(machineId)
	if err != nil {
		return nil, err
	}
	return &IDGenerator{
		snowflakeNode: node,
		redisClient:   redis,
		redisTimeout:  5 * time.Second,
	}, nil
}

func (g *IDGenerator) GenerateSeq(senderID, receiverID string) (int64, error) {
	key := fmt.Sprintf("seq:%s:%s", senderID, receiverID)
	ctx, cancel := context.WithTimeout(context.Background(), g.redisTimeout)
	defer cancel()

	seq, err := g.redisClient.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}

	return seq, nil
}

func (g *IDGenerator) GenerateMessageID() int64 {
	return g.snowflakeNode.Generate().Int64()
}

var (
	inst atomic.Pointer[IDGenerator]
	once sync.Once
)

func InitGenerator(machineId int64, redis *redis.ClusterClient) error {
	var err error
	once.Do(func() {
		generator, err := NewIDGenerator(machineId, redis)
		if err != nil {
			return
		}
		inst.Store(generator)
	})
	return err
}

func GeneratorFactory() *IDGenerator {
	return inst.Load()
}
