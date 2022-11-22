package executor

import (
	"os/exec"
	"sync"
	"time"

	"github.com/mneumi/etcd-crontab/common"
	"github.com/mneumi/etcd-crontab/worker/etcd"
)

var once sync.Once
var instance *executor

type executor struct {
	etcdInstance   etcd.IEtcd
	jobExecuteChan chan *common.JobExecuteInfo
	jobResultChan  chan *common.JobResult
}

func Start(etcdInstance etcd.IEtcd, jobExecuteChan chan *common.JobExecuteInfo, jobResultChan chan *common.JobResult) {
	once.Do(func() {
		initExecutor(etcdInstance, jobExecuteChan, jobResultChan)
		go instance.loop()
	})
}

func initExecutor(etcdInstance etcd.IEtcd,
	jobExecuteChan chan *common.JobExecuteInfo,
	jobResultChan chan *common.JobResult) {

	instance = &executor{}

	instance.etcdInstance = etcdInstance
	instance.jobExecuteChan = jobExecuteChan
	instance.jobResultChan = jobResultChan
}

func (e *executor) loop() {
	for jobInfo := range e.jobExecuteChan {
		e.execute(jobInfo)
	}
}

func (e *executor) execute(jobInfo *common.JobExecuteInfo) {
	go func() {
		job := jobInfo.Job

		cmd := exec.CommandContext(jobInfo.CancelContext, "/bin/bash", "-c", job.Command)

		startTime := time.Now()
		output, err := cmd.CombinedOutput()
		endTime := time.Now()

		result := &common.JobResult{
			Job:       job,
			Output:    output,
			Error:     err,
			StartTime: startTime,
			EndTime:   endTime,
		}

		e.jobResultChan <- result
	}()
}
