package config

import "testing"

func TestXxx(t *testing.T) {
	t.Log("\n [INFO] redis node:", GatewayCfg.Application.Component.Redis.Nodes)
	t.Log("\n [INFO] consul 相关配置文件:", GatewayCfg.Application.Component.Consul.Service.Tags)
}
