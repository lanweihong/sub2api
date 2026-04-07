package repository

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss/credentials"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

// OSSBackupStore implements service.BackupObjectStore using Alibaba Cloud OSS
type OSSBackupStore struct {
	client *oss.Client
	bucket string
	tmpDir string
}

// NewOSSBackupStore creates an OSSBackupStore from BackupStorageConfig
func NewOSSBackupStore(ctx context.Context, cfg *service.BackupStorageConfig) (*OSSBackupStore, error) {
	endpoint := cfg.OSSEndpoint
	if endpoint == "" {
		return nil, fmt.Errorf("oss_endpoint is required for OSS provider")
	}

	region := cfg.OSSRegion
	if region == "" {
		region = "cn-hangzhou"
	}

	credProvider := credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, "")

	ossCfg := oss.LoadDefaultConfig().
		WithCredentialsProvider(credProvider).
		WithEndpoint(endpoint).
		WithRegion(region)

	client := oss.NewClient(ossCfg)

	return &OSSBackupStore{client: client, bucket: cfg.Bucket, tmpDir: cfg.TempDir}, nil
}

func (o *OSSBackupStore) Upload(ctx context.Context, key string, body io.Reader, contentType string) (int64, error) {
	// 将数据写入临时文件，避免大文件完全加载到内存
	tmpFile, size, err := spoolToTempFile(body, o.tmpDir)
	if err != nil {
		return 0, err
	}
	defer func() {
		_ = tmpFile.Close()
		_ = os.Remove(tmpFile.Name())
	}()

	_, err = o.client.PutObject(ctx, &oss.PutObjectRequest{
		Bucket:      &o.bucket,
		Key:         &key,
		Body:        tmpFile,
		ContentType: &contentType,
	})
	if err != nil {
		return 0, fmt.Errorf("OSS PutObject: %w", err)
	}
	return size, nil
}

func (o *OSSBackupStore) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	result, err := o.client.GetObject(ctx, &oss.GetObjectRequest{
		Bucket: &o.bucket,
		Key:    &key,
	})
	if err != nil {
		return nil, fmt.Errorf("OSS GetObject: %w", err)
	}
	return result.Body, nil
}

func (o *OSSBackupStore) Delete(ctx context.Context, key string) error {
	_, err := o.client.DeleteObject(ctx, &oss.DeleteObjectRequest{
		Bucket: &o.bucket,
		Key:    &key,
	})
	return err
}

func (o *OSSBackupStore) PresignURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	result, err := o.client.Presign(ctx, &oss.GetObjectRequest{
		Bucket: &o.bucket,
		Key:    &key,
	}, oss.PresignExpires(expiry))
	if err != nil {
		return "", fmt.Errorf("OSS presign url: %w", err)
	}
	return result.URL, nil
}

func (o *OSSBackupStore) HeadBucket(ctx context.Context) error {
	_, err := o.client.GetBucketInfo(ctx, &oss.GetBucketInfoRequest{
		Bucket: &o.bucket,
	})
	if err != nil {
		return fmt.Errorf("OSS GetBucketInfo failed: %w", err)
	}
	return nil
}
