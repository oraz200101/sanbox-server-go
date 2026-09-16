package database

import (
	"context"
	"fmt"
	"time"

	"github.com/oraz200101/sandbox-server/main/internal/config"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type MongoDBClient struct {
	client *mongo.Client
	db     *mongo.Database
}

func NewMongoDBClient(cfg *config.Config) (*MongoDBClient, error) {
	uri := fmt.Sprintf("mongodb://%s:%d", cfg.DBMongo.Host, cfg.DBMongo.Port)

	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx, nil); err != nil {
		return nil, err
	}

	db := client.Database(cfg.DBMongo.Name)

	return &MongoDBClient{client: client, db: db}, nil
}

func (m *MongoDBClient) Close() error {
	return nil
}
