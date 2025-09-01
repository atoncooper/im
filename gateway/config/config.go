package config

import (
	"log"
	"path/filepath"
	"sync"

	"github.com/atoncooper/im/utils"
	"github.com/spf13/viper"
)

var (
	GatewayCfg Config
	lock       sync.RWMutex
	updateChan = make(chan struct{}, 1)
)

type Config struct {
	Application Application `yaml:"application"`
}

type Application struct {
	Name      string          `yaml:"name"`
	Port      int             `yaml:"port"`
	Host      string          `yaml:"host"`
	Timeout   string          `yaml:"timeout"`
	NodeId    string          `yaml:"nodeId"`
	Tag       string          `yaml:"tag"`
	Cors      CorsConfig      `yaml:"cors"`
	Logger    LoggerConfig    `yaml:"logger"`
	Component ComponentConfig `yaml:"component"`
}

type CorsConfig struct {
	ContextPath    string `yaml:"contextPath"`
	AllowedOrigins string `yaml:"allowedOrigins"`
	AllowedMethods string `yaml:"allowedMethods"`
	AllowedHeaders string `yaml:"allowedHeaders"`
	MaxAge         int    `yaml:"maxAge"`
}

type LoggerConfig struct {
	Level  string     `yaml:"level"`
	Format string     `yaml:"format"`
	File   FileConfig `yaml:"file"`
}

type FileConfig struct {
	Path      string `yaml:"path"`
	MaxSize   int    `yaml:"maxSize"`
	MaxAge    int    `yaml:"maxAge"`
	MaxBackup int    `yaml:"maxBackups"`
}
type WebsocketConfig struct {
	ReadBufferSize   int    `yaml:"readerBufferSize"`
	WriteBufferSize  int    `yaml:"writeBufferSize"`
	MaxConn          int    `yaml:"maxConn"`
	HandshakeTimeout string `yaml:"handshakeTimeout"`
	PongWait         string `yaml:"pongWait"`
	PingPeriod       string `yaml:"pingPeriod"`
	WriteWait        string `yaml:"writeWait"`
}

type ComponentConfig struct {
	Consul Consul `yaml:"consul"`
	Redis  Redis  `yaml:"redis"`
	Kafka  Kafka  `yaml:"kafka"`
}

type Consul struct {
	Endpoint string    `yaml:"endpoint"`
	Port     int       `yaml:"port"`
	Scheme   string    `yaml:"scheme"`
	Timeout  string    `yaml:"timeout"`
	Service  ConsulSrv `yaml:"service"`
}

type ConsulSrv struct {
	Register    bool        `yaml:"register"`
	Tags        []string    `yaml:"tags"`
	HealthCheck HealthCheck `yaml:"healthCheck"`
}

type HealthCheck struct {
	Enable   bool   `yaml:"enable"`
	Interval int    `yaml:"interval"`
	Timeout  int    `yaml:"timeout"`
	HTTPPath string `yaml:"httpPath"`
}

type Redis struct {
	Nodes     string `yaml:"nodes"`
	Password  string `yaml:"password"`
	PoolSize  int    `yaml:"poolSize"`
	MinIdle   int    `yaml:"minIdle"`
	MaxIdle   int    `yaml:"maxIdle"`
	MaxActive int    `yaml:"maxActive"`
}

type Kafka struct {
	Brokers      string `yaml:"brokers"`
	Topic        string `yaml:"topic"`
	RequiredAcks int    `yaml:"requiredAcks"`
	MaxRetries   int    `yaml:"maxRetries"`
	GroupID      string `yaml:"groupId"`
	Dlq          Dlq    `yaml:"dlq"`
}

type Dlq struct {
	Enabled     bool   `yaml:"enabled"`
	TopicSuffix string `yaml:"topicSuffix"`
}

func init() {
	currDir, err := utils.GetWorkPath()
	if err != nil {
		panic(err)
	}

	v := viper.New()

	cfgPath := filepath.Join(currDir, "gateway.yaml")
	v.SetConfigFile(cfgPath)
	v.SetConfigType("yaml")

	// First loading
	if err = v.ReadInConfig(); err != nil {
		panic(err)
	}

	lock.Lock()
	defer lock.Unlock()
	_ = v.Unmarshal(&GatewayCfg)
	log.Println("[INFO] 加载配置文件完成")

}
