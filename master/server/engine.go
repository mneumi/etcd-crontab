package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mneumi/etcd-crontab/common"
	"github.com/mneumi/etcd-crontab/master/server/resp"
)

// 初始化Engine
func (s *server) initEngine() {
	e := gin.Default()

	// 注册路由
	v1Group := e.Group("v1")
	{
		v1Group.GET("/jobs", s.listJobs)
		v1Group.POST("/job", s.saveJob)
		v1Group.DELETE("/job/:name", s.deleteJob)
		v1Group.POST("/job/:name/kill", s.killJob)
	}

	s.engine = e
}

func (s *server) saveJob(ctx *gin.Context) {
	job := &common.Job{}

	err := ctx.BindJSON(&job)
	if err != nil {
		resp.Response(ctx, http.StatusBadRequest, resp.InvalidParams, "请检查参数")
		return
	}

	oldJob, err := s.jobManager.SaveJob(job)
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

	err := s.jobManager.DeleteJob(name)
	if err != nil {
		resp.Response(ctx, http.StatusInternalServerError, resp.ServerError, err.Error())
		return
	}

	resp.ResponseOK(ctx, "删除成功")
}

func (s *server) listJobs(ctx *gin.Context) {
	jobs, err := s.jobManager.ListJobs()

	if err != nil {
		resp.Response(ctx, http.StatusInternalServerError, resp.ServerError, err.Error())
		return
	}

	resp.ResponseOK(ctx, jobs)
}

func (s *server) killJob(ctx *gin.Context) {
	name := ctx.Param("name")

	if name == "" {
		resp.Response(ctx, http.StatusBadRequest, resp.InvalidParams, "请检查参数")
		return
	}

	err := s.jobManager.KillJob(name)
	if err != nil {
		resp.Response(ctx, http.StatusInternalServerError, resp.ServerError, err.Error())
		return
	}

	resp.ResponseOK(ctx, "强杀成功")
}
