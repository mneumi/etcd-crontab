package server

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mneumi/etcd-crontab/master/config"
	"github.com/mneumi/etcd-crontab/master/manager"
)

type server struct {
	engine  *gin.Engine
	manager *manager.Manager
	cfg     *config.Config
	workers []string
	index   int
	lock    sync.Mutex
}

func initServer() *server {
	s := &server{
		manager: manager.Initial(),
		cfg:     config.GetConfig(),
		workers: make([]string, 0),
	}
	s.initEngine()
	go s.watchWorkers()
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
