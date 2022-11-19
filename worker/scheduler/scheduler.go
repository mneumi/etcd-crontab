package scheduler

import (
	"fmt"
	"sync"
	"time"

	"github.com/mneumi/etcd-crontab/common"
	"github.com/mneumi/etcd-crontab/worker/executor"
)

// 单例模式
var once sync.Once
var instance *Scheduler

type Scheduler struct {
	jobEventChan   chan *common.JobEvent
	planTable      map[string]*common.JobSchedulerPlan
	executingTable map[string]*common.JobExecuteInfo
	executor       *executor.Executor
	jobResultChan  chan *common.JobExecuteResult
}

func New() *Scheduler {
	once.Do(func() {
		initScheduler()
	})
	return instance
}

func initScheduler() {
	jobResultChan := make(chan *common.JobExecuteResult, 1000)

	instance = &Scheduler{
		jobEventChan:   make(chan *common.JobEvent, 1000),
		planTable:      make(map[string]*common.JobSchedulerPlan),
		executingTable: make(map[string]*common.JobExecuteInfo),
		executor:       executor.New(jobResultChan),
		jobResultChan:  jobResultChan,
	}
}

func (s *Scheduler) Start() {
	// 启动调度协程
	instance.loop()
}

func (s *Scheduler) PushJobEvent(jobEvent *common.JobEvent) {
	s.jobEventChan <- jobEvent
}

// TrySchedule 尝试执行任务，并且返回离最近下一次任务时间执行的时间间隔
func (s *Scheduler) TrySchedule() time.Duration {
	timeNow := time.Now()
	nearTime := new(time.Time)

	// 如果调度表为空，则睡眠1秒，避免CPU空转
	if len(s.planTable) == 0 {
		return time.Second
	}

	// 遍历全部任务（速度很快）
	for _, jobPlan := range s.planTable {
		// 过期的任务立即尝试执行，并且计算下一次的调度时间
		if jobPlan.NextTime.Before(timeNow) || jobPlan.NextTime.Equal(timeNow) {
			// 尝试执行任务
			s.TryStartJob(jobPlan)
			// 计算下一次的调度时间
			jobPlan.NextTime = jobPlan.Expr.Next(timeNow)
		}

		// 统计最近的要过期的时间（N秒后执行）
		if nearTime == nil || jobPlan.NextTime.Before(*nearTime) {
			nearTime = &jobPlan.NextTime
		}
	}

	// 计算时间间隔
	scheduleAfter := (*nearTime).Sub(timeNow)

	return scheduleAfter
}

func (s *Scheduler) loop() {
	timer := time.NewTimer(0)

	for {
		select {
		case jobEvent := <-s.jobEventChan:
			s.handleJobEvent(jobEvent)
		case jobResult := <-s.jobResultChan:
			s.handleJobResult(jobResult)
		case <-timer.C:
		}
		// 计算调度时间
		scheduleAfter := s.TrySchedule()
		// 重置调度间隔
		timer.Reset(scheduleAfter)
	}
}

func (s *Scheduler) handleJobEvent(jobEvent *common.JobEvent) {
	jobName := jobEvent.Job.Name

	switch jobEvent.EventType {
	case common.EVENT_TYPE_SAVE:
		jobSchedulerPlan, err := common.NewJobSchedulerPlan(jobEvent.Job)
		if err != nil {
			return
		}
		s.planTable[jobName] = jobSchedulerPlan
	case common.EVENT_TYPE_DELETE:
		_, ok := s.planTable[jobName]
		if ok {
			delete(s.planTable, jobName)
		}
	}
}

func (s *Scheduler) handleJobResult(jobResult *common.JobExecuteResult) {
	fmt.Printf("处理任务结果: %s", jobResult.Output)
	// 删除执行任务
	delete(s.executingTable, jobResult.ExecuteInfo.Job.Name)
}

//////

func (s *Scheduler) TryStartJob(jobPlan *common.JobSchedulerPlan) {
	// 如果任务正在执行，跳过本次任务
	_, jobExecuting := s.executingTable[jobPlan.Job.Name]
	if jobExecuting {
		fmt.Println("尚未退出，跳过执行任务")
		return
	}

	// 构建执行状态信息
	jobExecuteInfo := common.NewJobExecuteInfo(jobPlan)

	// 保存执行状态
	s.executingTable[jobPlan.Job.Name] = jobExecuteInfo

	// 执行任务
	// fmt.Println("执行任务：", jobExecuteInfo.Job.Name, jobExecuteInfo.PlanTime, jobExecuteInfo.RealTime)
	s.executor.ExecuteJob(jobExecuteInfo)
}

func (s *Scheduler) PushJobResult(jobResult *common.JobExecuteResult) {
	s.jobResultChan <- jobResult
}
