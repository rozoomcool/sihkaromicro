package kafka

import (
	"context"

	"github.com/rozoomcool/sihkaromicro/sources/internal/config"
	"github.com/segmentio/kafka-go"
)

type Producer struct {
	writer *kafka.Writer
}

func NewProducer(cfg config.KafkaConfig) *Producer {
	writer := &kafka.Writer{
		Addr:                   kafka.TCP(cfg.Brokers...),
		Balancer:               &kafka.LeastBytes{},
		AllowAutoTopicCreation: true, // автоматически создаёт topic если не существует
	}
	return &Producer{writer: writer}
}

// Publish — универсальный метод, topic передаётся явно
func (p *Producer) Publish(ctx context.Context, topic, key, payload string) error {
	return p.writer.WriteMessages(ctx, kafka.Message{
		Topic: topic,
		Key:   []byte(key),
		Value: []byte(payload),
	})
}

func (p *Producer) Close() error {
	return p.writer.Close()
}
