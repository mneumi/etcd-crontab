package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mneumi/etcd-crontab/master/config"
)

// 初始化Engine
func initEngine() *gin.Engine {
	e := gin.Default()

	// 注册路由
	v1Group := e.Group("v1")
	{
		v1Group.GET("/job/save", func(ctx *gin.Context) {})
	}

	return e
}

// 启动服务
func Start() {
	cfg := config.GetConfig()

	engine := initEngine()

	s := &http.Server{
		Handler:      engine,
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
	}

	s.ListenAndServe()
}
