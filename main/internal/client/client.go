package client

import (
	"github.com/oraz200101/sandbox-server/main/internal/database"
	"github.com/oraz200101/sandbox-server/main/internal/messaging"
)

type Client struct {
	KafkaClient         *messaging.KafkaClient
	MongoDBClient       *database.MongoDBClient
	PostgresClient      *database.PostgresClient
	ElasticSearchClient *database.ElasticSearchClient
}
