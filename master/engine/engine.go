package engine

import (
	"math/rand"
	"net/http"
	"strconv"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/mneumi/etcd-crontab/common"
	"github.com/mneumi/etcd-crontab/master/manager"
	"github.com/mneumi/etcd-crontab/master/middleware"
	"github.com/mneumi/etcd-crontab/master/resp"
)

var instance *engine

type engine struct {
	ge      *gin.Engine
	m       manager.IManager
	workers []string
	index   int
	lock    sync.Mutex
}

func init() {
	initEngine()
}

func GetHandler() *gin.Engine {
	return instance.ge
}

func initEngine() {
	instance = &engine{}

	instance.ge = gin.Default()
	instance.m = manager.GetInstance()

	instance.registerMiddleware()
	instance.registerRoute()

	go instance.watchWorkers()
}

func (e *engine) registerMiddleware() {
	e.ge.Use(middleware.GetCORS())
	e.ge.Use(middleware.GetIntercept())
}

func (e *engine) registerRoute() {
	// 注册路由
	v1Group := e.ge.Group("v1")
	{
		v1Group.GET("/jobs", e.listJobs)
		v1Group.POST("/job", e.saveJob)
		v1Group.DELETE("/job/:name", e.deleteJob)
		v1Group.POST("/job/:name/abort", e.abortJob)
		v1Group.GET("/workers", e.listWorkers)
		v1Group.GET("/job/log", e.listJobLog)
	}
}

func (e *engine) saveJob(ctx *gin.Context) {
	job := &common.Job{}

	err := ctx.BindJSON(&job)
	if err != nil {
		resp.Response(ctx, http.StatusBadRequest, resp.InvalidParams, "请检查参数")
		return
	}

	workerID := e.calculateLoadBalance(job)
	job.WorkerID = workerID

	oldJob, err := e.m.SaveJob(job)
	if err != nil {
		resp.Response(ctx, http.StatusInternalServerError, resp.ServerError, err.Error())
		return
	}

	resp.ResponseOK(ctx, oldJob)
}

func (e *engine) deleteJob(ctx *gin.Context) {
	name := ctx.Param("name")
	if name == "" {
		resp.Response(ctx, http.StatusBadRequest, resp.InvalidParams, "请检查参数")
		return
	}

	err := e.m.DeleteJob(name)
	if err != nil {
		resp.Response(ctx, http.StatusInternalServerError, resp.ServerError, err.Error())
		return
	}

	resp.ResponseOK(ctx, "删除成功")
}

func (e *engine) listJobs(ctx *gin.Context) {
	jobs, err := e.m.ListJobs()

	if err != nil {
		resp.Response(ctx, http.StatusInternalServerError, resp.ServerError, err.Error())
		return
	}

	resp.ResponseOK(ctx, jobs)
}

func (e *engine) abortJob(ctx *gin.Context) {
	name := ctx.Param("name")
	if name == "" {
		resp.Response(ctx, http.StatusBadRequest, resp.InvalidParams, "请检查参数")
		return
	}

	err := e.m.AbortJob(name)
	if err != nil {
		resp.Response(ctx, http.StatusInternalServerError, resp.ServerError, err.Error())
		return
	}

	resp.ResponseOK(ctx, "终止成功")
}

func (e *engine) listWorkers(ctx *gin.Context) {
	workers, err := e.m.ListWorkers()

	if err != nil {
		resp.Response(ctx, http.StatusInternalServerError, resp.ServerError, err.Error())
		return
	}

	resp.ResponseOK(ctx, workers)
}

func (e *engine) listJobLog(ctx *gin.Context) {
	name := ctx.Query("name")
	skipParam := ctx.DefaultQuery("skip", "0")
	limitParm := ctx.DefaultQuery("limit", "10")

	skip, err := strconv.ParseInt(skipParam, 10, 64)
	if err != nil {
		resp.Response(ctx, http.StatusBadRequest, resp.InvalidParams, "请检查参数")
		return
	}

	limit, err := strconv.ParseInt(limitParm, 10, 64)
	if err != nil {
		resp.Response(ctx, http.StatusBadRequest, resp.InvalidParams, "请检查参数")
		return
	}

	logs, err := e.m.ListJobLogs(name, skip, limit)
	if err != nil {
		resp.Response(ctx, http.StatusInternalServerError, resp.ServerError, err.Error())
		return
	}

	resp.ResponseOK(ctx, logs)
}

func (e *engine) calculateLoadBalance(job *common.Job) string {
	workerID := ""

	switch job.LoadBalance {
	case "random": // 随机
		e.lock.Lock()
		defer e.lock.Unlock()
		randIndex := rand.Intn(len(e.workers))
		workerID = e.workers[randIndex]
	case "robin": // 轮询
		e.lock.Lock()
		defer e.lock.Unlock()
		e.index = (e.index + 1) % len(e.workers)
		workerID = e.workers[e.index]
	default: // 指定机器
		workerID = job.LoadBalance
	}

	return workerID
}

func (e *engine) watchWorkers() {
	initWorkers, addWorkerChan, delWorkerChan := e.m.WatchWorkers()
	// 处理 Master 启动前，已经存在的 Worker
	e.workers = initWorkers
	// 监听后续变化
	for {
		select {
		case addWorkerID := <-addWorkerChan:
			e.lock.Lock()
			e.workers = append(e.workers, addWorkerID)
			e.lock.Unlock()
		case delWorkerID := <-delWorkerChan:
			e.lock.Lock()
			e.workers = common.DeleteElement(e.workers, delWorkerID)
			e.lock.Unlock()
		}
	}
}
