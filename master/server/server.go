package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mneumi/etcd-crontab/master/config"
	"github.com/mneumi/etcd-crontab/master/jobmanager"
)

type server struct {
	engine     *gin.Engine
	jobManager *jobmanager.JobManager
	cfg        *config.Config
}

func initServer() *server {
	s := &server{
		jobManager: jobmanager.New(),
		cfg:        config.GetConfig(),
	}
	s.initEngine()

	return s
}

// 启动服务
func Start() {
	server := initServer()
	hs := &http.Server{
		Handler:      server.engine,
		Addr:         fmt.Sprintf(":%d", server.cfg.Server.Port),
		ReadTimeout:  time.Duration(server.cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(server.cfg.Server.WriteTimeout) * time.Second,
	}

	hs.ListenAndServe()
}
