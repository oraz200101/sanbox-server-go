package database

import (
	"fmt"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/oraz200101/sandbox-server/main/internal/config"
)

type ElasticSearchClient struct {
	client *elasticsearch.Client
}

func NewElasticsearchClient(cfg *config.Config) (*ElasticSearchClient, error) {
	esConfig := elasticsearch.Config{
		Addresses: []string{
			fmt.Sprintf("http://%s:%d", cfg.Elasticsearch.Host, cfg.Elasticsearch.Port),
		},
	}

	client, err := elasticsearch.NewClient(esConfig)
	if err != nil {
		return nil, err
	}

	_, err = client.Info()
	if err != nil {
		return nil, err
	}

	return &ElasticSearchClient{client: client}, nil
}

func (e *ElasticSearchClient) GetClient() *elasticsearch.Client {
	return e.client
}

func (e *ElasticSearchClient) Close() error {
	return nil
}
