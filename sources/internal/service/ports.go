package service

import (
	"context"
	"io"
)

type StorageService interface {
	Upload(ctx context.Context, objectName string, reader io.Reader, size int64, contentType string) error
	Delete(ctx context.Context, objectName string) error
	ObjectName(ownerID string, sourceID int64, fileName string) string
	PresignedURL(ctx context.Context, objectName string) (string, error)
}

type MessageProducer interface {
	Publish(ctx context.Context, topic, key, payload string) error
	Close() error
}

type ProjectsClient interface {
	CheckAccess(ctx context.Context, projectID int64) (bool, error)
}
