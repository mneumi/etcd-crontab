package jobmanager

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/mneumi/etcd-crontab/common"
	"github.com/mneumi/etcd-crontab/master/config"
	"github.com/mneumi/etcd-crontab/worker/scheduler"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// 单例对象
var once sync.Once
var instance *JobManager

type JobManager struct {
	client    *clientv3.Client
	kv        clientv3.KV
	lease     clientv3.Lease
	watcher   clientv3.Watcher
	watchChan clientv3.WatchChan

	scheduler *scheduler.Scheduler
}

func New() *JobManager {
	once.Do(func() {
		initJobManager()
	})
	return instance
}

func (j *JobManager) Start() {
	// 启动调度协程
	go j.scheduler.Start()

	// 启动监听协程
	go j.start()
}

func initJobManager() {
	cfg := config.GetConfig()

	etcdConfig := clientv3.Config{
		Endpoints:   cfg.Etcd.Endpoints,
		DialTimeout: time.Duration(cfg.Etcd.DialTimeout) * time.Millisecond,
	}

	client, err := clientv3.New(etcdConfig)
	if err != nil {
		log.Fatalln(err)
	}

	for _, endpoint := range cfg.Etcd.Endpoints {
		timeoutCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		_, err := client.Status(timeoutCtx, endpoint)
		if err != nil {
			log.Fatalln(err)
		}
	}

	instance = &JobManager{}
	instance.client = client
	instance.kv = clientv3.NewKV(client)
	instance.lease = clientv3.NewLease(client)
	instance.watcher = clientv3.NewWatcher(client)

	instance.scheduler = scheduler.New()
}

func (j *JobManager) start() error {
	getResp, err := j.kv.Get(context.Background(), common.JOB_SAVE_DIR, clientv3.WithPrefix())
	if err != nil {
		return err
	}

	// 1.处理 Worker 启动前已经存在的 Job
	for _, kvPair := range getResp.Kvs {
		// 反序列化得到 Job
		job := common.NewJob()
		err := job.Unmarshal(kvPair.Value)
		if err != nil {
			return err
		}
		// 构建 JobEvent
		jobEvent := common.NewJobEvent(common.EVENT_TYPE_SAVE, job)
		// 推送事件给 Scheduler
		j.scheduler.PushJobEvent(jobEvent)
	}

	// 2.启动监听协程，处理后续新增的 Job
	watchStartRevision := getResp.Header.Revision
	go j.watchJobs(watchStartRevision)

	return nil
}

func (j *JobManager) watchJobs(revision int64) {
	// 开启监听，并保存监听 Channel 到 JobManager 中
	watchChan := j.watcher.Watch(context.Background(), common.JOB_SAVE_DIR,
		clientv3.WithRev(revision), clientv3.WithPrefix())
	j.watchChan = watchChan

	// 不断循环，处理监听事件
	for watchResp := range watchChan {
		for _, watchEvent := range watchResp.Events {
			jobEvent := new(common.JobEvent)
			switch watchEvent.Type {
			case mvccpb.PUT: // 任务保存事件
				// 反序列化，得到 job 对象
				job := common.NewJob()
				err := job.Unmarshal(watchEvent.Kv.Value)
				if err != nil {
					continue
				}
				// 创建事件
				jobEvent = common.NewJobEvent(common.EVENT_TYPE_SAVE, job)
			case mvccpb.DELETE: // 任务删除事件
				// 从 Key 中提取 Job 名称
				jobName := common.ExtraceJobNameByKey(watchEvent.Kv.String())
				// 创建事件
				jobEvent = common.NewJobEvent(common.EVENT_TYPE_DELETE, &common.Job{Name: jobName})
			}
			// 推送事件给 Scheduler
			j.scheduler.PushJobEvent(jobEvent)
		}
	}
}
