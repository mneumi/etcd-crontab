package executor

import (
	"context"
	"os/exec"
	"sync"
	"time"

	"github.com/mneumi/etcd-crontab/common"
)

var once sync.Once
var instance *Executor

type Executor struct {
	JobResultChan chan *common.JobExecuteResult
}

func (e *Executor) ExecuteJob(info *common.JobExecuteInfo) {
	go func() {
		// 获取分布式锁

		cmd := exec.CommandContext(context.Background(), "/bin/bash", "-c", info.Job.Command)

		startTime := time.Now()
		output, err := cmd.CombinedOutput()
		endTime := time.Now()

		result := &common.JobExecuteResult{
			ExecuteInfo: info,
			Output:      output,
			Error:       err,
			StartTime:   startTime,
			EndTime:     endTime,
		}

		e.JobResultChan <- result
	}()
}

func New(JobResultChan chan *common.JobExecuteResult) *Executor {
	once.Do(func() {
		initExecutor(JobResultChan)
	})
	return instance
}

func initExecutor(jobResultChan chan *common.JobExecuteResult) {
	instance = &Executor{
		JobResultChan: jobResultChan,
	}
}
