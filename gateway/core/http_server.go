package core

import (
	"fmt"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type HttpConfig struct {
	Host           string
	Port           int
	Timeout        string
	ContextPath    string
	AllowedOrigins string
	AllowedMethods string
	AllowedHeaders string
	MaxAge         int
}

func newDefaultHttpConfig() *HttpConfig {
	return &HttpConfig{
		Host:           "localhost",
		Port:           8080,
		Timeout:        "10s",
		ContextPath:    "/",
		AllowedOrigins: "*",
		AllowedMethods: "GET,POST,PUT,DELETE,OPTIONS",
		AllowedHeaders: "Origin,Content-Type,Authorization",
		MaxAge:         12 * 3600,
	}
}

type HttpParams struct {
	defineParams  *HttpConfig
	defaultParams string
}

type HttpParamsOps func(*HttpParams)

func DefineParams(p *HttpConfig) HttpParamsOps {
	return func(params *HttpParams) {
		params.defineParams = p
	}
}

func DefaultParams(p string) HttpParamsOps {
	return func(params *HttpParams) {
		params.defaultParams = p
	}
}

func StartHttpServer(httpOpts ...HttpParamsOps) {
	params := &HttpParams{}
	for _, opt := range httpOpts {
		opt(params)
	}
	if params.defineParams == nil {
		params.defineParams = newDefaultHttpConfig()
	}

	cfg := params.defineParams

	// use gin to do http server
	engine := gin.Default()

	// cors
	engine.Use(cors.New(cors.Config{
		AllowOrigins:     []string{cfg.AllowedOrigins},
		AllowMethods:     []string{cfg.AllowedMethods},
		AllowHeaders:     []string{cfg.AllowedHeaders},
		MaxAge:           time.Duration(cfg.MaxAge) * time.Second,
		AllowCredentials: true,
	}))

	// Please write the route here
	engine.GET("/health", ping)
	engine.GET("/ws", websocketServer)

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	if err := engine.Run(addr); err != nil {
		panic(err)
	}
}

func ping(c *gin.Context) {
	c.JSON(200, gin.H{
		"message": "pong",
	})
}
