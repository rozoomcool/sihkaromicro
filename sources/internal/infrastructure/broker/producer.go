package broker

import (
	"context"

	"github.com/rozoomcool/sihkaromicro/sources/internal/config"
	kafka "github.com/segmentio/kafka-go"
)

type KafkaProducer struct {
	writer *kafka.Writer
}

func NewKafkaProducer(cfg config.KafkaConfig) *KafkaProducer {
	writer := &kafka.Writer{
		Addr:                   kafka.TCP(cfg.Brokers...),
		Balancer:               &kafka.LeastBytes{},
		AllowAutoTopicCreation: true,
	}
	return &KafkaProducer{writer: writer}
}

func (p *KafkaProducer) Publish(ctx context.Context, topic, key, payload string) error {
	return p.writer.WriteMessages(ctx, kafka.Message{
		Topic: topic,
		Key:   []byte(key),
		Value: []byte(payload),
	})
}

func (p *KafkaProducer) Close() error {
	return p.writer.Close()
}
