package logger

import (
	"context"
	"sync"

	"github.com/mneumi/etcd-crontab/common"
	"github.com/mneumi/etcd-crontab/worker/mongodb"
	"go.mongodb.org/mongo-driver/mongo"
)

var once sync.Once
var instance *logger

type logger struct {
	collection *mongo.Collection
	jobLogChan chan *common.JobLog
}

func Start() chan *common.JobLog {
	once.Do(func() {
		initLogger()
		go instance.loop()
	})
	return instance.jobLogChan
}

func initLogger() {
	instance = &logger{}

	m := mongodb.GetInstance()
	collection := m.GetCollection()

	instance.collection = collection
	instance.jobLogChan = make(chan *common.JobLog, 1000)
}

func (l *logger) loop() {
	for record := range l.jobLogChan {
		l.collection.InsertOne(context.Background(), record)
	}
}
