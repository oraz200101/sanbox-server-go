package client

import (
	"github.com/oraz200101/sandbox-server/main/internal/broker"
	"github.com/oraz200101/sandbox-server/main/internal/database"
)

type Client struct {
	KafkaClient    *broker.KafkaClient
	MongoDBClient  *database.MongoDBClient
	PostgresClient *database.PostgresClient
}
