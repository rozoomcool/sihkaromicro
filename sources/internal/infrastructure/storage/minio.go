package storage

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/rozoomcool/sihkaromicro/sources/internal/config"
)

type MinioClient struct {
	client     *minio.Client
	bucketName string
}

func NewMinioClient(cfg config.MinIOConfig) (*MinioClient, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, err
	}

	return &MinioClient{client: client, bucketName: cfg.BucketName}, nil
}

func (c *MinioClient) EnsureBucket(ctx context.Context) error {
	exists, err := c.client.BucketExists(ctx, c.bucketName)
	if err != nil {
		return err
	}
	if !exists {
		return c.client.MakeBucket(ctx, c.bucketName, minio.MakeBucketOptions{})
	}
	return nil
}

func (c *MinioClient) Upload(ctx context.Context, objectName string, reader io.Reader, size int64, contentType string) error {
	_, err := c.client.PutObject(ctx, c.bucketName, objectName, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	return err
}

func (c *MinioClient) Delete(ctx context.Context, objectName string) error {
	return c.client.RemoveObject(ctx, c.bucketName, objectName, minio.RemoveObjectOptions{})
}

func (c *MinioClient) ObjectName(ownerID string, sourceID int64, fileName string) string {
	return fmt.Sprintf("%s/%d/%s", ownerID, sourceID, fileName)
}

func (c *MinioClient) PresignedURL(ctx context.Context, objectName string) (string, error) {
	url, err := c.client.PresignedGetObject(ctx, c.bucketName, objectName, time.Hour, nil)
	if err != nil {
		return "", err
	}
	return url.String(), nil
}
