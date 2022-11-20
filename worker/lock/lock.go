package lock

import (
	"context"
	"errors"
	"fmt"

	"github.com/mneumi/etcd-crontab/common"
	"github.com/mneumi/etcd-crontab/worker/etcd"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type jobLock struct {
	etcdInstance etcd.IEtcd
	jobName      string
	leaseID      clientv3.LeaseID
	cancelFunc   context.CancelFunc
}

func New(jobName string, etcdInstance etcd.IEtcd) *jobLock {
	return &jobLock{
		etcdInstance: etcdInstance,
		jobName:      jobName,
	}
}

func (j *jobLock) TryLock() error {
	kv := j.etcdInstance.GetKv()
	lease := j.etcdInstance.GetLease()

	// 创建租约
	leaseResp, err := lease.Grant(context.Background(), 5)
	if err != nil {
		return err
	}

	// 开启自动续租
	ctx, cancel := context.WithCancel(context.Background())
	leaseID := leaseResp.ID

	keepAliveChan, err := lease.KeepAlive(ctx, leaseID)
	if err != nil {
		j.revokeLease(leaseID, cancel)
		return err
	}

	// 开启协程处理 keepAliveChan
	go func() {
		for range keepAliveChan {
			// 消费 keepAliveChan 消息，避免 ETCD 发出警告
		}
	}()

	// 创建事务
	txn := kv.Txn(context.Background())
	lockKey := fmt.Sprintf("%s/%s", common.JOB_LOCK_DIR, j.jobName)

	// 通过事务抢锁
	txn.If(clientv3.Compare(clientv3.CreateRevision(lockKey), "=", 0)).
		Then(clientv3.OpPut(lockKey, "", clientv3.WithLease(leaseID))).
		Else() // 如果失败，则啥也不干

	txnResp, err := txn.Commit()
	// 如果事务提交失败，直接释放租约
	if err != nil {
		j.revokeLease(leaseID, cancel)
		return err
	}

	// 抢锁失败，直接返回
	if !txnResp.Succeeded {
		j.revokeLease(leaseID, cancel)
		return errors.New("抢锁失败")
	}

	// 抢锁成功，将 leaseID 和 cancel 方法注入到锁中
	j.leaseID = leaseID
	j.cancelFunc = cancel

	return nil
}

func (j *jobLock) Unlock() {
	// 如果抢到锁，才有会下列两个属性
	if j.leaseID != 0 && j.cancelFunc != nil {
		j.revokeLease(j.leaseID, j.cancelFunc)
	}
}

func (j *jobLock) revokeLease(leaseID clientv3.LeaseID, cancel context.CancelFunc) {
	lease := j.etcdInstance.GetLease()

	cancel()                                    // 取消自动续约协程
	lease.Revoke(context.Background(), leaseID) // 主动释放租约
}
