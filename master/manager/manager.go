package manager

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/mneumi/etcd-crontab/common"
	"github.com/mneumi/etcd-crontab/master/etcd"
	"github.com/mneumi/etcd-crontab/master/mongodb"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var once sync.Once
var instance *manager

type IManager interface {
	SaveJob(job *common.Job) (*common.Job, error)
	DeleteJob(name string) error
	ListJobs() ([]*common.Job, error)
	AbortJob(name string) error
	ListWorkers() ([]*common.Worker, error)
	WatchWorkers() ([]string, chan string, chan string)
	WatchIntercept() chan bool
	ListJobLogs(name string, skip int64, limit int64) ([]*common.JobLog, error)
}

type manager struct {
	client     *clientv3.Client
	kv         clientv3.KV
	lease      clientv3.Lease
	watcher    clientv3.Watcher
	collection *mongo.Collection
}

func GetInstance() *manager {
	once.Do(func() {
		initManager()
	})
	return instance
}

func initManager() {
	instance = &manager{}

	etcdInstance := etcd.GetInstance()
	mongoDBInstance := mongodb.GetInstance()

	instance.client = etcdInstance.GetClient()
	instance.kv = etcdInstance.GetKv()
	instance.lease = etcdInstance.GetLease()
	instance.watcher = etcdInstance.GetWatcher()
	instance.collection = mongoDBInstance.GetCollection()
}

func (m *manager) SaveJob(job *common.Job) (*common.Job, error) {
	jobKey := getJobKey(job.Name)

	jobValue := job.Marshal()

	putResp, err := m.kv.Put(context.Background(), jobKey, string(jobValue), clientv3.WithPrevKV())
	if err != nil {
		return job, err
	}

	// 如果更新，返回旧值
	if putResp.PrevKv != nil {
		oldJob := common.NewJob()
		oldJob.Unmarshal(putResp.PrevKv.Value)

		return oldJob, nil
	}

	return job, nil
}

func (m *manager) DeleteJob(name string) error {
	jobKey := getJobKey(name)

	_, err := m.kv.Delete(context.Background(), jobKey)
	if err != nil {
		return err
	}

	return nil
}

func (m *manager) ListJobs() ([]*common.Job, error) {
	jobs := []*common.Job{}

	getResp, err := m.kv.Get(context.Background(), common.JOB_SAVE_DIR, clientv3.WithPrefix())
	if err != nil {
		return jobs, err
	}

	for _, kvPair := range getResp.Kvs {
		job := common.NewJob()

		err := job.Unmarshal(kvPair.Value)
		if err != nil {
			return jobs, err
		}

		jobs = append(jobs, job)
	}

	return jobs, nil
}

func (m *manager) AbortJob(name string) error {
	jobKey := getAbortJobKey(name)
	ttl := 10

	lease, err := m.lease.Grant(context.Background(), int64(ttl))
	if err != nil {
		return err
	}
	_, err = m.kv.Put(context.Background(), jobKey, name, clientv3.WithLease(lease.ID))
	if err != nil {
		return err
	}
	return nil
}

func (m *manager) ListWorkers() ([]*common.Worker, error) {
	workers := []*common.Worker{}

	getResp, err := m.kv.Get(context.Background(), common.WORKER_DIR, clientv3.WithPrefix())
	if err != nil {
		return workers, err
	}

	for _, kvPair := range getResp.Kvs {
		worker := &common.Worker{}

		err := worker.Unmarshal(kvPair.Value)
		if err != nil {
			return workers, err
		}

		workers = append(workers, worker)
	}

	return workers, nil
}

func (m *manager) WatchWorkers() ([]string, chan string, chan string) {
	addWorkerChan := make(chan string, 100)
	delWorkerChan := make(chan string, 100)
	initWorkers := make([]string, 0)

	// 处理现有的 Worker
	getResp, err := m.kv.Get(context.Background(), common.WORKER_DIR, clientv3.WithPrefix())
	if err != nil {
		log.Fatalln("获取Worker节点失败")
	}
	for _, kvs := range getResp.Kvs {
		workerID := common.ExtraceWorkerIDByKey(string(kvs.Key))
		initWorkers = append(initWorkers, workerID)
	}

	// 监听后续添加的 Worker
	go func() {
		watchChan := m.watcher.Watch(context.Background(), common.WORKER_DIR, clientv3.WithPrefix())
		for watchResp := range watchChan {
			for _, event := range watchResp.Events {
				workerID := common.ExtraceWorkerIDByKey(string(event.Kv.Key))
				switch event.Type {
				case mvccpb.PUT:
					addWorkerChan <- workerID
				case mvccpb.DELETE:
					delWorkerChan <- workerID
				}
			}
		}
	}()

	return initWorkers, addWorkerChan, delWorkerChan
}

func (m *manager) WatchIntercept() chan bool {
	interceptChan := make(chan bool, 10)

	// 监听KV变化
	go func() {
		watchChan := m.watcher.Watch(context.Background(), common.INTERCEPT_DIR, clientv3.WithPrefix())
		for watchResp := range watchChan {
			for _, event := range watchResp.Events {
				// 默认情况下开启拦截模式
				// 只有显式在 etcd 中设置 /intercept/mode = disable 才能关闭拦截模式
				intercept := true

				switch event.Type {
				case mvccpb.PUT:
					key := common.ExtractInterceptNameByKey(string(event.Kv.Key))
					value := string(event.Kv.Value)
					if key == common.INTERCEPT_KEY && value == common.INTERCEPT_DISABLE {
						intercept = false
					}
				case mvccpb.DELETE:
					intercept = true
				}

				interceptChan <- intercept
			}
		}
	}()

	return interceptChan
}

func (m *manager) ListJobLogs(name string, skip int64, limit int64) ([]*common.JobLog, error) {
	logs := []*common.JobLog{}

	filter := &common.JobLogFilter{
		JobName: name,
	}
	logSort := &common.SortLogByStartTime{
		SortOrder: -1,
	}

	cursor, err := m.collection.Find(context.Background(), filter,
		options.Find().SetSort(logSort),
		options.Find().SetSkip(skip),
		options.Find().SetLimit(limit))
	if err != nil {
		return logs, err
	}
	defer cursor.Close(context.Background())

	for cursor.Next(context.Background()) {
		jobLog := &common.JobLog{}

		err := cursor.Decode(&jobLog)
		if err != nil {
			continue
		}

		logs = append(logs, jobLog)
	}

	return logs, nil
}

func getJobKey(name string) string {
	return fmt.Sprintf("%s%s", common.JOB_SAVE_DIR, name)
}

func getAbortJobKey(name string) string {
	return fmt.Sprintf("%s%s", common.JOB_ABORT_DIR, name)
}
