package config

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/hashicorp/consul/api"
)

var serverNodeId string

// 定义错误变量
var (
	ErrServiceNotFound = errors.New("service not found in consul")
	ErrNoHealthyNodes  = errors.New("no healthy nodes available for service")
)

type ConsulConf struct {
	Address    string
	Scheme     string
	Datacenter string
}

type ServerMeta struct {
	ID      string
	Name    string
	Host    string
	Port    int
	Tag     string
	Check   string
	Timeout time.Duration
}

type ConsulClient struct {
	client *api.Client
	conf   *ConsulConf
	mutex  sync.Mutex
}

var (
	consulClient *ConsulClient
	clientOnce   sync.Once
)

func newConsulClient(conf *ConsulConf) (*ConsulClient, error) {
	if conf == nil {
		return nil, errors.New("consul config is nil")
	}
	clientOnce.Do(func() {
		cfg := api.DefaultConfig()
		cfg.Address = conf.Address
		cfg.Scheme = conf.Scheme
		cfg.Datacenter = conf.Datacenter

		client, err := api.NewClient(cfg)
		if err == nil {
			consulClient = &ConsulClient{
				client: client,
				conf:   conf,
			}
		}
	})
	if consulClient == nil {
		return nil, fmt.Errorf("failed to initialize consul client")
	}
	return consulClient, nil
}

func NewConsul(conf *ConsulConf, meta ServerMeta) error {

	client, err := newConsulClient(conf)
	if err != nil {
		return err
	}

	check := &api.AgentServiceCheck{}

	switch meta.Check {
	case "http":
		check.HTTP = fmt.Sprintf("http://%s:%d/health", meta.Host, meta.Port)
	case "tcp":
		check.TCP = fmt.Sprintf("%s:%d", meta.Host, meta.Port)
	default:
		return fmt.Errorf("unsupported check type: %s", meta.Check)
	}
	check.Timeout = meta.Timeout.String()
	check.Interval = "10s"
	check.DeregisterCriticalServiceAfter = "30s"

	reg := &api.AgentServiceRegistration{
		ID:      meta.ID,
		Name:    meta.Name,
		Tags:    []string{meta.Tag},
		Address: meta.Host,
		Port:    meta.Port,
		Check:   check,
	}

	if err := client.client.Agent().ServiceRegister(reg); err != nil {
		return fmt.Errorf("register service: %w", err)
	}

	serverNodeId = meta.ID

	return nil
}

func GetNodeIdTemplate() string {
	return serverNodeId
}

func (c *ConsulClient) GetServiceAddTemplate(ctx context.Context,
	serviceName string,
) ([]string, error) {
	if ctx == nil {
		defaultCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		ctx = defaultCtx
	}

	// 查询健康服务实例
	services, _, err := c.client.Health().Service(serviceName, "", true, &api.QueryOptions{})
	if err != nil {
		return nil, fmt.Errorf("query service %s failed: %w", serviceName, err)
	}
	if len(services) == 0 {
		return nil, ErrNoHealthyNodes
	}

	// 构建服务实例列表
	var serviceAddrs []string
	for _, service := range services {
		serviceAddrs = append(serviceAddrs, fmt.Sprintf("%s:%d", service.Service.Address, service.Service.Port))
	}
	return serviceAddrs, nil
}

// GetServiceInstanceTemplate 获取服务的所有健康实例地址
func (c *ConsulClient) GetServiceInstanceTemplate(
	ctx context.Context,
	serviceName string,
) ([]string, error) {
	if ctx == nil {
		// 设置超时
		defaultCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		ctx = defaultCtx
	}

	services, _, err := c.client.Health().Service(serviceName, "", true, &api.QueryOptions{})
	if err != nil {
		return nil, fmt.Errorf("query service %s failed: %w", serviceName, err)
	}

	if len(services) == 0 {
		return nil, ErrNoHealthyNodes
	}

	// 收集所有健康实例的地址
	addresses := make([]string, 0, len(services))
	for _, s := range services {
		addresses = append(addresses, fmt.Sprintf("%s:%d", s.Service.Address, s.Service.Port))
	}

	return addresses, nil
}
