package server

import (
	"math/rand"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/mneumi/etcd-crontab/common"
	"github.com/mneumi/etcd-crontab/master/server/resp"
)

// 初始化Engine
func (s *server) initEngine() {
	s.engine = gin.Default()

	// 注册路由
	v1Group := s.engine.Group("v1")
	{
		v1Group.GET("/jobs", s.listJobs)
		v1Group.POST("/job", s.saveJob)
		v1Group.DELETE("/job/:name", s.deleteJob)
		v1Group.POST("/job/:name/abort", s.abortJob)
		v1Group.GET("/workers", s.listWorkers)
		v1Group.GET("/job/log", s.listJobLog)
	}
}

func (s *server) saveJob(ctx *gin.Context) {
	job := &common.Job{}

	err := ctx.BindJSON(&job)
	if err != nil {
		resp.Response(ctx, http.StatusBadRequest, resp.InvalidParams, "请检查参数")
		return
	}

	workerID := s.calculateLoadBalance(job)
	job.WorkerID = workerID

	oldJob, err := s.manager.SaveJob(job)
	if err != nil {
		resp.Response(ctx, http.StatusInternalServerError, resp.ServerError, err.Error())
		return
	}

	resp.ResponseOK(ctx, oldJob)
}

func (s *server) deleteJob(ctx *gin.Context) {
	name := ctx.Param("name")

	if name == "" {
		resp.Response(ctx, http.StatusBadRequest, resp.InvalidParams, "请检查参数")
		return
	}

	err := s.manager.DeleteJob(name)
	if err != nil {
		resp.Response(ctx, http.StatusInternalServerError, resp.ServerError, err.Error())
		return
	}

	resp.ResponseOK(ctx, "删除成功")
}

func (s *server) listJobs(ctx *gin.Context) {
	jobs, err := s.manager.ListJobs()

	if err != nil {
		resp.Response(ctx, http.StatusInternalServerError, resp.ServerError, err.Error())
		return
	}

	resp.ResponseOK(ctx, jobs)
}

func (s *server) abortJob(ctx *gin.Context) {
	name := ctx.Param("name")

	if name == "" {
		resp.Response(ctx, http.StatusBadRequest, resp.InvalidParams, "请检查参数")
		return
	}

	err := s.manager.AbortJob(name)
	if err != nil {
		resp.Response(ctx, http.StatusInternalServerError, resp.ServerError, err.Error())
		return
	}

	resp.ResponseOK(ctx, "终止成功")
}

func (s *server) listWorkers(ctx *gin.Context) {
	workers, err := s.manager.ListWorkers()

	if err != nil {
		resp.Response(ctx, http.StatusInternalServerError, resp.ServerError, err.Error())
		return
	}

	resp.ResponseOK(ctx, workers)
}

func (s *server) listJobLog(ctx *gin.Context) {
	name := ctx.Query("name")
	skipParam := ctx.DefaultQuery("skip", "0")
	limitParm := ctx.DefaultQuery("limit", "20")

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

	logs, err := s.manager.ListJobLogs(name, skip, limit)
	if err != nil {
		resp.Response(ctx, http.StatusInternalServerError, resp.ServerError, err.Error())
		return
	}

	resp.ResponseOK(ctx, logs)
}

func (s *server) watchWorkers() {
	initWorkers, addWorkerChan, delWorkerChan := s.manager.WatchWorkers()
	// 处理 Master 启动前，已经存在的 Worker
	s.workers = initWorkers
	// 监听后续变化
	for {
		select {
		case addWorkerID := <-addWorkerChan:
			s.lock.Lock()
			s.workers = append(s.workers, addWorkerID)
			s.lock.Unlock()
		case delWorkerID := <-delWorkerChan:
			s.lock.Lock()
			s.workers = common.DeleteSliceByElement(s.workers, delWorkerID)
			s.lock.Unlock()
		}
	}
}

func (s *server) calculateLoadBalance(job *common.Job) string {
	workerID := ""

	switch job.LoadBalance {
	case "random": // 随机
		s.lock.Lock()
		defer s.lock.Unlock()
		randIndex := rand.Intn(len(s.workers))
		workerID = s.workers[randIndex]
	case "robin": // 轮询
		s.lock.Lock()
		defer s.lock.Unlock()
		s.index = (s.index + 1) % len(s.workers)
		workerID = s.workers[s.index]
	default: // 指定机器
		workerID = job.LoadBalance
	}

	return workerID
}
