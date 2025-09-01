package config

import (
	"fmt"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestConsulClient(t *testing.T) {

	ginRouter := gin.Default()
	ginRouter.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
		})
	})

	go func() {
		ginRouter.Run("127.0.0.1:8080")
	}()

	addr := fmt.Sprintf("%s:%d", GatewayCfg.Application.Component.Consul.Endpoint, GatewayCfg.Application.Component.Consul.Port)
	consulConf := &ConsulConf{
		Address: addr,
		Scheme:  GatewayCfg.Application.Component.Consul.Scheme,
	}
	meta := ServerMeta{
		ID:      GatewayCfg.Application.NodeId,
		Name:    GatewayCfg.Application.Name,
		Host:    GatewayCfg.Application.Host,
		Port:    GatewayCfg.Application.Port,
		Tag:     GatewayCfg.Application.Tag,
		Check:   "tcp",
		Timeout: time.Duration(10) * time.Second,
	}

	err := NewConsul(consulConf, meta)
	if err != nil {
		t.Fatalf("failed to register service: %v", err)
	}
	select {}
}
