package config

import (
	"log"
	"os"
	"sync"

	"gopkg.in/yaml.v2"
)

// 单例对象
var once sync.Once
var configInstance *Config

type Config struct {
	Server ServerConfig `yaml:"server"`
}

type ServerConfig struct {
	Port         int `yaml:"port"`
	ReadTimeout  int `yaml:"read_timeout"`
	WriteTimeout int `yaml:"write_timeout"`
}

// 获取配置对象
func GetConfig() *Config {
	once.Do(func() {
		file, err := os.ReadFile("config.yaml")
		if err != nil {
			log.Fatal(err)
		}

		err = yaml.Unmarshal(file, &configInstance)
		if err != nil {
			log.Fatal(err)
		}
	})

	return configInstance
}
