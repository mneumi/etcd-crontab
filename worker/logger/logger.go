package logger

import (
	"context"
	"fmt"
	"log"

	"github.com/mneumi/etcd-crontab/common"
	"github.com/mneumi/etcd-crontab/worker/config"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type logger struct {
	collection *mongo.Collection
	jobLogChan chan *common.JobLog
}

func Start() chan *common.JobLog {
	l := initLogger()
	go l.loop()

	return l.jobLogChan
}

func initLogger() *logger {
	cfg := config.GetConfig()
	collection := initCollection(cfg)

	return &logger{
		collection: collection,
		jobLogChan: make(chan *common.JobLog, 1000),
	}
}

func initCollection(cfg *config.Config) *mongo.Collection {
	uri := fmt.Sprintf("mongodb://%s", cfg.MongoDB.Host)

	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatalln(err)
	}

	// 选择数据库
	db := client.Database(cfg.MongoDB.DBName)

	// 选择表
	collection := db.Collection(cfg.MongoDB.Collection)

	return collection
}

func (l *logger) loop() {
	for record := range l.jobLogChan {
		_, err := l.collection.InsertOne(context.Background(), record)
		if err != nil {
			fmt.Println(err)
		}
	}
}
