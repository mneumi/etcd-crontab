package main

import (
	"fmt"
	"os"

	"github.com/mneumi/etcd-crontab/common"
	"github.com/mneumi/etcd-crontab/worker/etcd"
	"github.com/mneumi/etcd-crontab/worker/executor"
	"github.com/mneumi/etcd-crontab/worker/logger"
	"github.com/mneumi/etcd-crontab/worker/register"
	"github.com/mneumi/etcd-crontab/worker/scheduler"
	"github.com/mneumi/etcd-crontab/worker/watcher"
)

func main() {
	// 创建 Etcd 连接实例
	etcdInstance := etcd.GetInstance()

	// 注册 Woker 到 Etcd 中
	workID := os.Args[1]
	fmt.Println(workID)
	worker := common.NewWorkerWithIPv4(workID, workID)
	register.Registe(etcdInstance, worker)

	// 开启监听器
	jobEventChan := watcher.Start(etcdInstance)

	// 开启日志记录器
	logInfoChan := logger.Start()

	// 开启调度器
	jobExecuteChan, jobResultChan := scheduler.Start(worker, jobEventChan, logInfoChan)

	// 开启执行器
	executor.Start(etcdInstance, jobExecuteChan, jobResultChan)

	// 阻塞主goroutine，避免程序退出
	select {}
}
