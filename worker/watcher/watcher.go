package watcher

import (
	"context"
	"sync"

	"github.com/mneumi/etcd-crontab/common"
	"github.com/mneumi/etcd-crontab/worker/etcd"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
)

var once sync.Once
var instance *watcher

type watcher struct {
	etcdInstance etcd.IEtcd
	jobEventChan chan *common.JobEvent
}

func Start(etcdInstance etcd.IEtcd) chan *common.JobEvent {
	once.Do(func() {
		initWatcher(etcdInstance)
		go instance.loop()
	})
	return instance.jobEventChan
}

func initWatcher(etcdInstance etcd.IEtcd) {
	instance = &watcher{}

	instance.etcdInstance = etcdInstance
	instance.jobEventChan = make(chan *common.JobEvent, 1000)
}

func (w *watcher) loop() error {
	kv := w.etcdInstance.GetKv()

	// 1.处理 Worker 启动前已经存在的 Job
	getResp, err := kv.Get(context.Background(), common.JOB_SAVE_DIR, clientv3.WithPrefix())
	if err != nil {
		return err
	}

	for _, kvPair := range getResp.Kvs {
		// 反序列化得到 Job
		job := common.NewJob()
		err := job.Unmarshal(kvPair.Value)
		if err != nil {
			return err
		}
		// 构建 JobEvent
		jobEvent := common.NewJobEvent(common.EVENT_TYPE_SAVE, job)
		// 推送事件到 JobEventChan
		w.jobEventChan <- jobEvent
	}

	// 2.启动监听协程，处理后续新增的 Job
	revision := getResp.Header.Revision
	go w.watchJobs(revision)

	// 3.启动监听协程，处理终止指令
	go w.watchJobAbort()

	return nil
}

func (w *watcher) watchJobs(revision int64) {
	watcher := w.etcdInstance.GetWatcher()

	// 开启监听
	watchChan := watcher.Watch(context.Background(), common.JOB_SAVE_DIR,
		clientv3.WithRev(revision), clientv3.WithPrefix())

	// 不断循环，处理监听事件
	for watchResp := range watchChan {
		for _, event := range watchResp.Events {
			jobEvent := new(common.JobEvent)
			switch event.Type {
			case mvccpb.PUT: // 任务保存事件
				// 反序列化，得到 job 对象
				job := common.NewJob()
				err := job.Unmarshal(event.Kv.Value)
				if err != nil {
					continue
				}
				// 创建事件
				jobEvent = common.NewJobEvent(common.EVENT_TYPE_SAVE, job)
			case mvccpb.DELETE: // 任务删除事件
				// 从 Key 中提取 Job 名称
				jobName := common.ExtraceJobNameByKey(string(event.Kv.Key))
				// 创建事件
				jobEvent = common.NewJobEvent(common.EVENT_TYPE_DELETE, &common.Job{Name: jobName})
			}
			// 推送事件给 JobEventChan
			w.jobEventChan <- jobEvent
		}
	}
}

func (w *watcher) watchJobAbort() {
	watcher := w.etcdInstance.GetWatcher()

	// 开启监听
	watchChan := watcher.Watch(context.Background(), common.JOB_ABORT_DIR, clientv3.WithPrefix())

	// 不断循环，处理监听事件
	for watchResp := range watchChan {
		for _, event := range watchResp.Events {
			// jobEvent := new(common.JobEvent)
			switch event.Type {
			case mvccpb.PUT: // 新增终止任务事件
				// 反序列化，得到 job 对象
				job := &common.Job{
					Name: string(event.Kv.Value),
				}
				// 创建事件
				jobEvent := common.NewJobEvent(common.EVENT_TYPE_ABORT, job)
				// 推送事件给 JobEventChan
				w.jobEventChan <- jobEvent
			}
		}
	}
}
