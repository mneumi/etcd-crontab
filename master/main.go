package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/mneumi/etcd-crontab/master/config"
	"github.com/mneumi/etcd-crontab/master/engine"
)

func main() {
	cfg := config.GetConfig()
	handler := engine.GetHandler()

	server := &http.Server{
		Handler:      handler,
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
	}
	server.ListenAndServe()
}
