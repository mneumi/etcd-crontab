package mongodb

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/mneumi/etcd-crontab/worker/config"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var once sync.Once
var instance *mongoDB

type mongoDB struct {
	client     *mongo.Client
	db         *mongo.Database
	collection *mongo.Collection
}

func GetInstance() *mongoDB {
	once.Do(func() {
		initMongoDB()
	})
	return instance
}

func initMongoDB() {
	instance = &mongoDB{}

	cfg := config.GetConfig()

	uri := fmt.Sprintf("mongodb://%s", cfg.MongoDB.Host)

	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatalln(err)
	}

	db := client.Database(cfg.MongoDB.DBName)
	collection := db.Collection(cfg.MongoDB.Collection)

	instance.client = client
	instance.db = db
	instance.collection = collection
}

func (m *mongoDB) GetClient() *mongo.Client {
	return m.client
}

func (m *mongoDB) GetDB() *mongo.Database {
	return m.db
}

func (m *mongoDB) GetCollection() *mongo.Collection {
	return m.collection
}
