package broker

import (
	"fmt"

	"github.com/oraz200101/sandbox-server/main/internal/config"
	"github.com/oraz200101/sandbox-server/main/internal/domain"
	"github.com/segmentio/kafka-go"
)

type KafkaClient struct {
	writers map[string]*kafka.Writer
	reader  *kafka.Reader
}

func NewKafkaClient(cfg *config.Config) (*KafkaClient, error) {
	addr := fmt.Sprintf("%s:%d", cfg.Kafka.Host, cfg.Kafka.Port)

	writers := make(map[string]*kafka.Writer)

	writers[domain.TopicExams] = &kafka.Writer{
		Addr:  kafka.TCP(addr),
		Topic: domain.TopicExams,
	}

	writers[domain.TopicResults] = &kafka.Writer{
		Addr:  kafka.TCP(addr),
		Topic: domain.TopicResults,
	}

	writers[domain.TopicNotifications] = &kafka.Writer{
		Addr:  kafka.TCP(addr),
		Topic: domain.TopicNotifications,
	}

	return &KafkaClient{
		writers: writers,
	}, nil
}

func (k *KafkaClient) GetWriter(topic string) *kafka.Writer {
	return k.writers[topic]
}

func (k *KafkaClient) Close() error {
	for _, w := range k.writers {
		err := w.Close()
		if err != nil {
			return err
		}
	}
	return nil
}
