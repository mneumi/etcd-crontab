package main

import "github.com/mneumi/etcd-crontab/worker/jobmanager"

func main() {
	// 开启 JobManager
	jobmanager.New().Start()

	// 阻塞主goroutine，避免程序退出
	select {}
}
