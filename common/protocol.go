package common

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/gorhill/cronexpr"
)

type Job struct {
	Name           string `json:"name"`
	Command        string `json:"command"`
	CronExpression string `json:"cron_expression"`
}

func NewJob() *Job {
	return &Job{}
}

func (j *Job) Marshal() []byte {
	b, _ := json.Marshal(&j)
	return b
}

func (j *Job) Unmarshal(data []byte) error {
	return json.Unmarshal(data, &j)
}

func ExtraceJobNameByKey(jobKey string) string {
	return strings.TrimPrefix(jobKey, JOB_SAVE_DIR)
}

type EventType int

type JobEvent struct {
	EventType EventType // SAVE, DELETE
	Job       *Job
}

func NewJobEvent(eventType EventType, job *Job) *JobEvent {
	return &JobEvent{
		EventType: eventType,
		Job:       job,
	}
}

type JobSchedulerPlan struct {
	Job      *Job
	Expr     *cronexpr.Expression
	NextTime time.Time
}

func NewJobSchedulerPlan(job *Job) (*JobSchedulerPlan, error) {
	plan := &JobSchedulerPlan{
		Job: job,
	}

	// 解析 cron 表达式
	expr, err := cronexpr.Parse(job.CronExpression)
	if err != nil {
		return plan, err
	}

	// 生成调度计划
	plan.Expr = expr
	plan.NextTime = expr.Next(time.Now())

	return plan, nil
}

// 任务执行状态
type JobExecuteInfo struct {
	Job      *Job
	PlanTime time.Time
	RealTime time.Time
}

func NewJobExecuteInfo(plan *JobSchedulerPlan) *JobExecuteInfo {
	return &JobExecuteInfo{
		Job:      plan.Job,
		PlanTime: plan.NextTime,
		RealTime: time.Now(),
	}
}

type JobExecuteResult struct {
	ExecuteInfo *JobExecuteInfo
	Output      []byte
	Error       error
	StartTime   time.Time
	EndTime     time.Time
}
