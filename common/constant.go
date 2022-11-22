package common

const (
	JOB_SAVE_DIR  = "/cron/job/"
	JOB_ABORT_DIR = "/cron/abort/"
	JOB_LOCK_DIR  = "/cron/lock/"

	WORKER_DIR = "/cron/worker/"
)

const (
	EVENT_TYPE_SAVE   JobEventType = 1
	EVENT_TYPE_DELETE JobEventType = 2
	EVENT_TYPE_ABORT  JobEventType = 3
)

const (
	ENV_KEY_WORKER_ID = "WORKER_ID"
)

const (
	INTERCEPT_DIR     = "/intercept/"
	INTERCEPT_KEY     = "mode"
	INTERCEPT_DISABLE = "disable"
)
