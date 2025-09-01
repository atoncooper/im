package utils

import (
	"context"
	"sync"

	"github.com/redis/go-redis/v9"
)

// status
//
// 存放用户长连接相关记录状态
//
// 采用key + value 形式存储
// key : status:{uid}
//
// value : Meta

const PERFIX = "status:"

var (
	statusMap sync.Map

	statusIns  *status
	statusOnce sync.Once
)

type status struct {
	redis *redis.ClusterClient
}

type Meta struct {
	ServerId       string //  长连接服务器id
	ServerAddr     string // 长连接服务器地址
	UserRemoteAddr string // 用户远程地址
	Status         string // 用户状态 : online or deadline
}

func NewStatus(redis *redis.ClusterClient) *status {
	return &status{
		redis: redis,
	}
}

func InitStatus(redis *redis.ClusterClient) {
	statusOnce.Do(func() {
		statusIns = NewStatus(redis)
	})
}

func (s *status) InitStatus(
	uid string,
	meta Meta,
) error {
	key := PERFIX + uid

	_, err := s.redis.Set(context.Background(), key, meta, 0).Result()
	if err != nil {
		return err
	}

	statusMap.Store(uid, meta)

	return nil
}

func (s *status) ClearStatus(uid string) error {

	key := PERFIX + uid
	_, err := s.redis.Del(context.Background(), key).Result()
	if err != nil {
		return err
	}

	statusMap.Delete(uid)

	return nil
}

func (s *status) GetStatus(uid string) (Meta, error) {
	key := PERFIX + uid
	meta, ok := statusMap.Load(uid)
	if ok {
		return meta.(Meta), nil
	}
	meta, err := s.redis.Get(context.Background(), key).Result()
	if err != nil {
		return Meta{}, err
	}
	statusMap.Store(uid, meta)
	return meta.(Meta), nil
}

func StatusTemplate() *status {
	if statusIns == nil {
		panic("status: call InitStatus first")
	}
	return statusIns
}
