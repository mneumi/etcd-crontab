package etcd

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/mneumi/etcd-crontab/worker/config"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type IEtcd interface {
	GetClient() *clientv3.Client
	GetKv() clientv3.KV
	GetLease() clientv3.Lease
	GetWatcher() clientv3.Watcher
}

var once sync.Once
var instance *Etcd

type Etcd struct {
	Client  *clientv3.Client
	Kv      clientv3.KV
	Lease   clientv3.Lease
	Watcher clientv3.Watcher
}

func GetInstance() *Etcd {
	once.Do(func() {
		initEtcd()
	})
	return instance
}

func initEtcd() {
	cfg := config.GetConfig()

	etcdConfig := clientv3.Config{
		Endpoints:   cfg.Etcd.Endpoints,
		DialTimeout: time.Duration(cfg.Etcd.DialTimeout) * time.Millisecond,
	}

	client, err := clientv3.New(etcdConfig)
	if err != nil {
		log.Fatalln(err)
	}

	err = checkClient(client, cfg)
	if err != nil {
		log.Println("Etcd连接失败，请检查Etcd服务状态")
	}

	instance = &Etcd{}
	instance.Client = client
	instance.Kv = clientv3.NewKV(client)
	instance.Lease = clientv3.NewLease(client)
	instance.Watcher = clientv3.NewWatcher(client)
}

func checkClient(client *clientv3.Client, cfg *config.Config) error {
	for _, endpoint := range cfg.Etcd.Endpoints {
		timeoutCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		_, err := client.Status(timeoutCtx, endpoint)
		if err != nil {
			return err
		}
	}
	return nil
}

func (e *Etcd) GetClient() *clientv3.Client {
	return e.Client
}

func (e *Etcd) GetKv() clientv3.KV {
	return e.Kv
}

func (e *Etcd) GetLease() clientv3.Lease {
	return e.Lease
}

func (e *Etcd) GetWatcher() clientv3.Watcher {
	return e.Watcher
}
