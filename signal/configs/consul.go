package configs

import (
	"errors"
	"fmt"
	"time"

	"github.com/hashicorp/consul/api"
)

type ConsulClient struct {
	Client *api.Client
}

func NewEnterpriseClient(addrs []string, token, caFile string) (*api.Client, error) {
	if len(addrs) == 0 {
		return nil, errors.New("empty consul address list")
	}

	conf := api.DefaultConfig()
	conf.Token = token
	if caFile != "" {
		conf.TLSConfig = api.TLSConfig{
			CAFile:             caFile,
			InsecureSkipVerify: false,
		}
	}
	conf.HttpClient.Timeout = 5 * time.Second

	var lastErr error
	for _, addr := range addrs {
		conf.Address = addr
		return api.NewClient(conf)
	}
	return nil, fmt.Errorf("all consul addrs failed: %w", lastErr)
}

func (c *ConsulClient) RegisterService(serviceID, serviceName, address string,
	port int, tags []string, meta map[string]string,
) error {

	cfg := &api.AgentServiceRegistration{
		ID:      serviceID,
		Name:    serviceName,
		Address: address,
		Port:    port,
		Tags:    tags,
		Meta:    meta,
		Check: &api.AgentServiceCheck{
			TCP:      fmt.Sprintf("%s:%d", address, port),
			Interval: "10s",
			Timeout:  "2s",
		},
	}

	return c.Client.Agent().ServiceRegister(cfg)
}

func (c *ConsulClient) GetNodeID() (string, error) {
	self, err := c.Client.Agent().Self()
	if err != nil {
		return "", err
	}

	configMap, ok := self["Config"]
	if !ok {
		return "", errors.New("agent self: missing Config")
	}

	nodeID, ok := configMap["NodeID"].(string)
	if !ok || nodeID == "" {
		return "", errors.New("agent self: NodeID empty")
	}
	return nodeID, nil
}
