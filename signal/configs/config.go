package configs

import (
	"sync/atomic"
)

type Config struct {
	Application struct {
		Server struct {
			Host    string `mapstructure:"host"`
			Port    int    `mapstructure:"port"`
			Timeout int    `mapstructure:"timeout"`
		} `mapstructure:"server"`

		Component struct {
			Redis struct {
				Nodes []string `mapstructure:"nodes"`
			} `mapstructure:"redis"`
		} `mapstructure:"component"`
	} `mapstructure:"application"`
}

var Cfg atomic.Pointer[Config]

func Get() *Config {
	return Cfg.Load()
}
