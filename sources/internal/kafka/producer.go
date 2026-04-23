package kafka

import (
	"context"
	"encoding/json"

	"github.com/rozoomcool/sihkaromicro/sources/internal/config"
	"github.com/segmentio/kafka-go"
)

type ChunkingJob struct {
	SourceID  int64  `json:"source_id"`
	ProjectID int64  `json:"project_id"`
	OwnerID   string `json:"owner_id"`
	MinioPath string `json:"minio_path"`
	FileType  string `json:"file_type"`
	JobID     string `json:"job_id"`
}

type Producer struct {
	writer *kafka.Writer
}

func NewProducer(cfg config.KafkaConfig) *Producer {
	writer := &kafka.Writer{
		Addr:     kafka.TCP(cfg.Brokers...),
		Topic:    cfg.Topic,
		Balancer: &kafka.LeastBytes{},
	}
	return &Producer{writer: writer}
}

func (p *Producer) PublishChunkingJob(ctx context.Context, job ChunkingJob) error {
	payload, err := json.Marshal(job)
	if err != nil {
		return err
	}

	return p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(job.JobID),
		Value: payload,
	})
}

func (p *Producer) Close() error {
	return p.writer.Close()
}
