package config

import (
	"log"
	"os"
	"sync"

	"gopkg.in/yaml.v2"
)

// 单例对象
var once sync.Once
var instance *Config

type Config struct {
	Server ServerConfig `yaml:"server"`
	Etcd   EtcdConfig   `yaml:"etcd"`
}

type ServerConfig struct {
	Port         int `yaml:"port"`
	ReadTimeout  int `yaml:"read_timeout"`
	WriteTimeout int `yaml:"write_timeout"`
}

type EtcdConfig struct {
	Endpoints   []string `yaml:"endpoints"`
	DialTimeout int      `yaml:"dial_timeout"`
}

// 获取配置对象
func GetConfig() *Config {
	once.Do(func() {
		file, err := os.ReadFile("config.yaml")
		if err != nil {
			log.Fatal(err)
		}

		err = yaml.Unmarshal(file, &instance)
		if err != nil {
			log.Fatal(err)
		}
	})

	return instance
}
