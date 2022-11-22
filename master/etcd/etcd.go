package etcd

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/mneumi/etcd-crontab/worker/config"
	clientv3 "go.etcd.io/etcd/client/v3"
)

var once sync.Once
var instance *Etcd

type Etcd struct {
	client  *clientv3.Client
	kv      clientv3.KV
	lease   clientv3.Lease
	watcher clientv3.Watcher
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
		log.Fatalln("Etcd连接失败，请检查Etcd服务状态")
	}

	instance = &Etcd{}
	instance.client = client
	instance.kv = clientv3.NewKV(client)
	instance.lease = clientv3.NewLease(client)
	instance.watcher = clientv3.NewWatcher(client)
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
	return e.client
}

func (e *Etcd) GetKv() clientv3.KV {
	return e.kv
}

func (e *Etcd) GetLease() clientv3.Lease {
	return e.lease
}

func (e *Etcd) GetWatcher() clientv3.Watcher {
	return e.watcher
}
