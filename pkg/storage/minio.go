package storage

import (
	"context"
	"fmt"

	"github.com/manyodream/gonetdisk/configs"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func NewMinioClient(cfg configs.MinioConfig) (*minio.Client, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, err
	}

	exists, err := client.BucketExists(context.Background(), cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("检查 Bucket 失败: %w", err)
	}
	if !exists {
		err = client.MakeBucket(
			context.Background(),
			cfg.Bucket,
			minio.MakeBucketOptions{Region: "us-east-1"},
		)
		if err != nil {
			return nil, fmt.Errorf("创建 Bucket 失败: %w", err)
		}
	}

	return client, nil
}
