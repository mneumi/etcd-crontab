package register

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/mneumi/etcd-crontab/common"
	"github.com/mneumi/etcd-crontab/worker/etcd"
	clientv3 "go.etcd.io/etcd/client/v3"
)

var once sync.Once
var instance *register

type register struct {
	etcdInstance etcd.IEtcd
	worker       *common.Worker
	leaseID      clientv3.LeaseID
	cancel       context.CancelFunc
}

func Registe(etcdInstance etcd.IEtcd, worker *common.Worker) {
	once.Do(func() {
		initRegiter(etcdInstance, worker)
		instance.registe()
	})
}

func initRegiter(etcdInstance etcd.IEtcd, worker *common.Worker) {
	instance = &register{}

	instance.etcdInstance = etcdInstance
	instance.worker = worker
}

func (r *register) registe() {
	kv := r.etcdInstance.GetKv()
	lease := r.etcdInstance.GetLease()

	regKey := fmt.Sprintf("%s%s", common.WORKER_DIR, r.worker.ID)

	// 创建租约
	leaseResp, err := lease.Grant(context.Background(), 10)
	if err != nil {
		log.Fatalln("注册服务失败 [创建租约失败]: ", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	leaseID := leaseResp.ID

	// 注入到 register 中
	r.leaseID = leaseID
	r.cancel = cancel

	// 开启自动续租
	keepAliveChan, err := lease.KeepAlive(ctx, leaseID)
	if err != nil {
		r.revokeLease()
		log.Fatalln("注册服务失败 [自动续租失败]: ", err)
	}

	// 开启协程处理 keepAliveChan
	go func() {
		for range keepAliveChan {
			// 消费 keepAliveChan 消息，避免 Etcd 发出警告
		}
	}()

	// 注册到 Etcd
	_, err = kv.Put(ctx, regKey, string(r.worker.Marshal()), clientv3.WithLease(leaseID))
	if err != nil {
		r.revokeLease()
		log.Fatalln("注册服务失败: ", err)
	}
}

func (r *register) revokeLease() {
	if r.leaseID != 0 && r.cancel != nil {
		lease := r.etcdInstance.GetLease()
		r.cancel()                                    // 取消自动续约协程
		lease.Revoke(context.Background(), r.leaseID) // 主动释放租约
	}
}
