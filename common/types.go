package common

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/gorhill/cronexpr"
)

type Job struct {
	Name           string `json:"name"`
	Command        string `json:"command"`
	CronExpression string `json:"cron_expression"`
	LoadBalance    string `json:"load_balance"`
	WorkerID       string `json:"worker_id"`
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

type JobSchedulePlan struct {
	Job      *Job
	Expr     *cronexpr.Expression
	NextTime time.Time
}

func NewJobSchedulePlan(job *Job) (*JobSchedulePlan, error) {
	plan := &JobSchedulePlan{
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

type JobExecuteResult struct {
	Job       *Job
	Output    []byte
	Error     error
	StartTime time.Time
	EndTime   time.Time
}

type JobLog struct {
	Name           string `bson:"name" json:"name"`
	Command        string `bson:"command" json:"command"`
	CronExpression string `bson:"cron_expression" json:"cron_expression"`
	LoadBalance    string `bson:"load_balance" json:"load_balance"`
	WorkerID       string `bson:"worker_id" json:"worker_id"`
	Result         string `bson:"result" json:"result"`
	Error          string `bson:"error" json:"error"`
	StartTimestamp int64  `bson:"start_timestamp" json:"start_timestamp"`
	EndTimestamp   int64  `bson:"end_timestamp" json:"end_timestamp"`
}

type JobLogFilter struct {
	JobName string `bson:"name"`
}

type SortLogByStartTime struct {
	SortOrder int `bson:"start_timestamp"`
}

type JobExecuteInfo struct {
	Job           *Job
	CancelContext context.Context
	CancelFunc    context.CancelFunc
}

func NewJobExecuteInfo(job *Job) *JobExecuteInfo {
	ctx, cancel := context.WithCancel(context.Background())
	return &JobExecuteInfo{
		Job:           job,
		CancelContext: ctx,
		CancelFunc:    cancel,
	}
}

type Worker struct {
	ID   string `json:"id"`
	IP   string `json:"ip"`
	Name string `json:"name"`
}

func NewWorker(id string, ip string, name string) *Worker {
	return &Worker{
		ID:   id,
		IP:   ip,
		Name: name,
	}
}

func NewWorkerWithIPv4(id string, name string) *Worker {
	ip, err := GetIPv4()
	if err != nil {
		ip = err.Error()
	}
	return NewWorker(id, ip, name)
}

func (w *Worker) Marshal() []byte {
	b, _ := json.Marshal(&w)
	return b
}

func (w *Worker) Unmarshal(data []byte) error {
	return json.Unmarshal(data, &w)
}

func ExtraceJobNameByKey(key string) string {
	return strings.TrimPrefix(key, JOB_SAVE_DIR)
}

func ExtraceWorkerIDByKey(key string) string {
	return strings.TrimPrefix(key, WORKER_DIR)
}
