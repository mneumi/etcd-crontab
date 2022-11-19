package jobmanager

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/mneumi/etcd-crontab/common"
	"github.com/mneumi/etcd-crontab/master/config"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// 单例对象
var once sync.Once
var instance *JobManager

type JobManager struct {
	client *clientv3.Client
	kv     clientv3.KV
	lease  clientv3.Lease
}

func New() *JobManager {
	once.Do(func() {
		initJobManager()
	})

	return instance
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
}

func (j *JobManager) SaveJob(job *common.Job) (*common.Job, error) {
	jobKey := getJobKey(job.Name)

	jobValue := job.Marshal()

	putResp, err := j.kv.Put(context.Background(), jobKey, string(jobValue), clientv3.WithPrevKV())
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

func (j *JobManager) DeleteJob(name string) error {
	jobKey := getJobKey(name)

	_, err := j.kv.Delete(context.Background(), jobKey)
	if err != nil {
		return err
	}

	return nil
}

func (j *JobManager) ListJobs() ([]*common.Job, error) {
	jobs := []*common.Job{}
	dirKey := common.JOB_SAVE_DIR

	getResp, err := j.kv.Get(context.Background(), dirKey, clientv3.WithPrefix())
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

func (j *JobManager) KillJob(name string) error {
	jobKey := getKillJobKey(name)
	ttl := 10

	lease, err := j.lease.Grant(context.Background(), int64(ttl))
	if err != nil {
		return err
	}

	_, err = j.kv.Put(context.Background(), jobKey, name, clientv3.WithLease(lease.ID))
	if err != nil {
		return err
	}

	return nil
}

func getJobKey(name string) string {
	return fmt.Sprintf("%s/%s", common.JOB_SAVE_DIR, name)
}

func getKillJobKey(name string) string {
	return fmt.Sprintf("%s/%s", common.JOB_KILL_DIR, name)
}
