package executor

import (
	"os/exec"
	"time"

	"github.com/mneumi/etcd-crontab/common"
	"github.com/mneumi/etcd-crontab/worker/etcd"
)

type executor struct {
	etcdInstance   etcd.IEtcd
	jobExecuteChan chan *common.JobExecuteInfo
	jobResultChan  chan *common.JobExecuteResult
}

func Start(etcdInstance etcd.IEtcd, jobExecuteChan chan *common.JobExecuteInfo, jobResultChan chan *common.JobExecuteResult) {
	e := inital(etcdInstance, jobExecuteChan, jobResultChan)
	go e.loop()
}

func inital(etcdInstance etcd.IEtcd, jobExecuteChan chan *common.JobExecuteInfo, jobResultChan chan *common.JobExecuteResult) *executor {
	instance := &executor{
		etcdInstance:   etcdInstance,
		jobExecuteChan: jobExecuteChan,
		jobResultChan:  jobResultChan,
	}
	return instance
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

		result := &common.JobExecuteResult{
			Job:       job,
			Output:    output,
			Error:     err,
			StartTime: startTime,
			EndTime:   endTime,
		}

		e.jobResultChan <- result
	}()
}
