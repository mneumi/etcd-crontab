package main

import (
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
	worker := common.NewWorkerWithIPv4()
	register.Registe(etcdInstance, worker)

	// 开启监听器
	jobEventChan := watcher.Start(etcdInstance)

	// 开启日志记录器
	jobLogChan := logger.Start()

	// 开启调度器
	jobExecuteChan, jobResultChan := scheduler.Start(worker, jobEventChan, jobLogChan)

	// 开启执行器
	executor.Start(etcdInstance, jobExecuteChan, jobResultChan)

	select {}
}
