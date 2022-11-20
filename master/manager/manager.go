package manager

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/mneumi/etcd-crontab/common"
	"github.com/mneumi/etcd-crontab/master/config"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Manager struct {
	client     *clientv3.Client
	kv         clientv3.KV
	lease      clientv3.Lease
	watcher    clientv3.Watcher
	collection *mongo.Collection
}

func Initial() *Manager {
	m := &Manager{}
	m = connectEtcd(m)
	m = connectMongoDB(m)
	return m
}

func connectEtcd(m *Manager) *Manager {
	cfg := config.GetConfig()

	// 连接 Etcd
	etcdConfig := clientv3.Config{
		Endpoints:   cfg.Etcd.Endpoints,
		DialTimeout: time.Duration(cfg.Etcd.DialTimeout) * time.Millisecond,
	}

	etcdClient, err := clientv3.New(etcdConfig)
	if err != nil {
		log.Fatalln(err)
	}

	for _, endpoint := range cfg.Etcd.Endpoints {
		timeoutCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		_, err := etcdClient.Status(timeoutCtx, endpoint)
		if err != nil {
			log.Fatalln(err)
		}
	}

	m.client = etcdClient
	m.kv = clientv3.NewKV(etcdClient)
	m.lease = clientv3.NewLease(etcdClient)
	m.watcher = clientv3.NewWatcher(etcdClient)

	return m
}

func connectMongoDB(m *Manager) *Manager {
	cfg := config.GetConfig()

	// 连接 MongoDB，获取 Collection
	uri := fmt.Sprintf("mongodb://%s", cfg.MongoDB.Host)

	mongoClient, err := mongo.Connect(context.Background(), options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatalln(err)
	}

	db := mongoClient.Database(cfg.MongoDB.DBName)
	collection := db.Collection(cfg.MongoDB.Collection)

	m.collection = collection

	return m
}

func (m *Manager) SaveJob(job *common.Job) (*common.Job, error) {
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

func (m *Manager) DeleteJob(name string) error {
	jobKey := getJobKey(name)

	_, err := m.kv.Delete(context.Background(), jobKey)
	if err != nil {
		return err
	}

	return nil
}

func (m *Manager) ListJobs() ([]*common.Job, error) {
	jobs := []*common.Job{}
	dirKey := common.JOB_SAVE_DIR

	getResp, err := m.kv.Get(context.Background(), dirKey, clientv3.WithPrefix())
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

func (m *Manager) AbortJob(name string) error {
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

func (m *Manager) ListJobLogs(name string, skip int64, limit int64) ([]*common.JobLog, error) {
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
		fmt.Println(name, skip, limit)

		jobLog := &common.JobLog{}

		err := cursor.Decode(&jobLog)
		if err != nil {
			continue
		}

		logs = append(logs, jobLog)
	}

	return logs, nil
}

func (m *Manager) ListWorkers() ([]*common.Worker, error) {
	workers := []*common.Worker{}
	dirKey := common.WORKER_DIR

	getResp, err := m.kv.Get(context.Background(), dirKey, clientv3.WithPrefix())
	if err != nil {
		return workers, err
	}

	for _, kvPair := range getResp.Kvs {
		worker := common.NewWorker("", "", "")

		err := worker.Unmarshal(kvPair.Value)
		if err != nil {
			return workers, err
		}

		workers = append(workers, worker)
	}

	return workers, nil
}

func (m *Manager) WatchWorkers() ([]string, chan string, chan string) {
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

func getJobKey(name string) string {
	return fmt.Sprintf("%s%s", common.JOB_SAVE_DIR, name)
}

func getAbortJobKey(name string) string {
	return fmt.Sprintf("%s%s", common.JOB_ABORT_DIR, name)
}
