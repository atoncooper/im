package utils

import (
	"sync"
	"time"
)

// 服务实例信息缓存
// 采用sync.map 进行缓存 并定期刷新 以及变更通知
// 定期刷新默认时间间隔为10s一次

type ServiceMeta struct {
	ServiceName string
	InstanceID  string
	Address     string
	Port        int
	Tags        []string
}

type ServiceInstanceCache struct {
	InstanceMap     sync.Map
	RefreshInterval time.Duration
	ChangeChan      chan struct{}
	stopChan        chan struct{}
}

func NewServiceInstanceCache(serviceName string, refreshInterval time.Duration) *ServiceInstanceCache {
	return &ServiceInstanceCache{
		InstanceMap:     sync.Map{},
		RefreshInterval: refreshInterval,
		ChangeChan:      make(chan struct{}),
		stopChan:        make(chan struct{}),
	}
}
