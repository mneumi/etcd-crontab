package scheduler

import (
	"sync"
	"time"

	"github.com/mneumi/etcd-crontab/common"
)

var once sync.Once
var instance *scheduler

type scheduler struct {
	worker            *common.Worker
	jobEventChan      chan *common.JobEvent
	jobExecuteChan    chan *common.JobExecuteInfo
	jobResultChan     chan *common.JobResult
	jobLogChan        chan *common.JobLog
	schedulePlanTable map[string]*common.JobSchedulePlan
	executingTable    map[string]*common.JobExecuteInfo
}

func Start(worker *common.Worker, jobEventChan chan *common.JobEvent, jobLogChan chan *common.JobLog) (chan *common.JobExecuteInfo, chan *common.JobResult) {
	once.Do(func() {
		initScheduler(worker, jobEventChan, jobLogChan)
		go instance.loop()
	})
	return instance.jobExecuteChan, instance.jobResultChan
}

func initScheduler(worker *common.Worker, jobEventChan chan *common.JobEvent, jobLogChan chan *common.JobLog) {
	instance = &scheduler{}

	instance.worker = worker
	instance.jobEventChan = jobEventChan
	instance.jobLogChan = jobLogChan
	instance.jobExecuteChan = make(chan *common.JobExecuteInfo, 1000)
	instance.jobResultChan = make(chan *common.JobResult, 1000)
	instance.schedulePlanTable = make(map[string]*common.JobSchedulePlan)
	instance.executingTable = make(map[string]*common.JobExecuteInfo)
}

func (s *scheduler) loop() {
	timer := time.NewTimer(0)

	for {
		select {
		case jobEvent := <-s.jobEventChan:
			s.handleJobEvent(jobEvent)
		case jobResult := <-s.jobResultChan:
			s.handleJobResult(jobResult)
		case <-timer.C:
		}
		// 进行调度
		waitTime := s.schedule()
		// 重置调度间隔
		timer.Reset(waitTime)
	}
}

func (s *scheduler) handleJobEvent(jobEvent *common.JobEvent) {
	jobName := jobEvent.Job.Name
	workerID := jobEvent.Job.WorkerID

	switch jobEvent.EventType {
	case common.EVENT_TYPE_SAVE:
		// 如果 jobName 在调度表中，但 wokerID 不同，说明任务变更了执行者
		if _, ok := s.schedulePlanTable[jobName]; ok && workerID != s.worker.ID {
			delete(s.schedulePlanTable, jobName)
		}
		// 如果 workerID 相同，则加入到调度表中
		if s.worker.ID == workerID {
			jobSchedulePlan, err := common.NewJobSchedulePlan(jobEvent.Job)
			if err != nil {
				return
			}
			s.schedulePlanTable[jobName] = jobSchedulePlan
		}
	case common.EVENT_TYPE_DELETE:
		// 从调度表中删除
		delete(s.schedulePlanTable, jobName)
	case common.EVENT_TYPE_ABORT:
		// 处理终止任务
		if jobExecuteInfo, ok := s.executingTable[jobName]; ok {
			jobExecuteInfo.CancelFunc()
		}
	}
}

func (s *scheduler) handleJobResult(jobResult *common.JobResult) {
	// 从执行表中删除任务
	delete(s.executingTable, jobResult.Job.Name)

	// 处理结果中的错误信息
	errInfo := ""
	if jobResult.Error != nil {
		errInfo = jobResult.Error.Error()
	}

	// 推送到日志协程
	jobLog := &common.JobLog{
		Name:           jobResult.Job.Name,
		Command:        jobResult.Job.Command,
		CronExpression: jobResult.Job.CronExpression,
		LoadBalance:    jobResult.Job.LoadBalance,
		WorkerID:       jobResult.Job.WorkerID,
		Result:         string(jobResult.Output),
		Error:          errInfo,
		StartTimestamp: jobResult.StartTime.Unix(),
		EndTimestamp:   jobResult.EndTime.Unix(),
	}
	s.jobLogChan <- jobLog
}

// 尝试执行任务，并且返回离最近下一次任务时间执行的时间间隔
func (s *scheduler) schedule() time.Duration {
	timeNow := time.Now()
	nearTime := new(time.Time)

	// 如果调度表为空，则返回休眠时间1秒，避免CPU空转
	if len(s.schedulePlanTable) == 0 {
		return time.Second
	}

	// 遍历调度表中全部任务
	for _, jobPlan := range s.schedulePlanTable {
		// 过期的任务立即尝试执行，并且计算下一次的调度时间
		if jobPlan.NextTime.Before(timeNow) || jobPlan.NextTime.Equal(timeNow) {
			// 尝试执行 Job
			s.startJob(jobPlan.Job)
			// 计算下一次的调度时间
			jobPlan.NextTime = jobPlan.Expr.Next(timeNow)
		}

		// 统计最近的要过期的时间（休眠 N 秒后执行）
		if nearTime == nil || jobPlan.NextTime.Before(*nearTime) {
			nearTime = &jobPlan.NextTime
		}
	}

	// 计算时间间隔
	waitTime := (*nearTime).Sub(timeNow)
	return waitTime
}

func (s *scheduler) startJob(job *common.Job) {
	// 如果任务正在执行，跳过本次任务
	_, executing := s.executingTable[job.Name]
	if executing {
		return
	}

	// 构建任务执行信息
	jobExecuteInfo := common.NewJobExecuteInfo(job)

	// 保存任务到执行表中
	s.executingTable[job.Name] = jobExecuteInfo

	// 推送任务到执行Chan
	s.jobExecuteChan <- jobExecuteInfo
}
