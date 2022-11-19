package common

import "encoding/json"

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
