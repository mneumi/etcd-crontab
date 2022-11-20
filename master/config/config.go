package config

import (
	"log"
	"os"
	"sync"

	"gopkg.in/yaml.v2"
)

var once sync.Once
var instance *Config

type Config struct {
	Server  ServerConfig  `yaml:"server"`
	Etcd    EtcdConfig    `yaml:"etcd"`
	MongoDB MongoDBConfig `yaml:"mongodb"`
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

type MongoDBConfig struct {
	Host        string `yaml:"host"`
	DialTimeout int    `yaml:"dial_timeout"`
	DBName      string `yaml:"db_name"`
	Collection  string `yaml:"collection"`
}

// 获取配置对象
func GetConfig() *Config {
	once.Do(func() {
		initConfig()
	})
	return instance
}

func initConfig() {
	file, err := os.ReadFile("config.yaml")
	if err != nil {
		log.Fatal(err)
	}

	err = yaml.Unmarshal(file, &instance)
	if err != nil {
		log.Fatal(err)
	}
}
