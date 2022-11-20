package common

const (
	JOB_SAVE_DIR  = "/cron/job/"
	JOB_ABORT_DIR = "/cron/abort/"
	JOB_LOCK_DIR  = "/cron/lock/"

	WORKER_DIR = "/cron/worker/"
)

const (
	EVENT_TYPE_SAVE   EventType = 1
	EVENT_TYPE_DELETE EventType = 2
	EVENT_TYPE_ABORT  EventType = 3
)
