package repository

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/qiniu/go-sdk/v7/auth"
	"github.com/qiniu/go-sdk/v7/storage"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

// QiniuBackupStore implements service.BackupObjectStore using Qiniu Kodo
type QiniuBackupStore struct {
	mac       *auth.Credentials
	bucket    string
	domain    string
	region    *storage.Region
	bucketMgr *storage.BucketManager
	tmpDir    string
}

// NewQiniuBackupStore creates a QiniuBackupStore from BackupStorageConfig
func NewQiniuBackupStore(_ context.Context, cfg *service.BackupStorageConfig) (*QiniuBackupStore, error) {
	if cfg.QiniuDomain == "" {
		return nil, fmt.Errorf("qiniu_domain is required for Qiniu provider")
	}

	mac := auth.New(cfg.AccessKeyID, cfg.SecretAccessKey)
	region := parseQiniuRegion(cfg.QiniuRegion)

	bucketCfg := storage.Config{
		Region: region,
	}
	bucketMgr := storage.NewBucketManager(mac, &bucketCfg)

	return &QiniuBackupStore{
		mac:       mac,
		bucket:    cfg.Bucket,
		domain:    cfg.QiniuDomain,
		region:    region,
		bucketMgr: bucketMgr,
		tmpDir:    cfg.TempDir,
	}, nil
}

func (q *QiniuBackupStore) Upload(ctx context.Context, key string, body io.Reader, _ string) (int64, error) {
	// 将数据写入临时文件，避免大文件完全加载到内存
	tmpFile, size, err := spoolToTempFile(body, q.tmpDir)
	if err != nil {
		return 0, err
	}
	defer func() {
		_ = tmpFile.Close()
		_ = os.Remove(tmpFile.Name())
	}()

	putPolicy := storage.PutPolicy{
		Scope:   fmt.Sprintf("%s:%s", q.bucket, key), // 允许覆盖上传
		Expires: 3600,
	}
	upToken := putPolicy.UploadToken(q.mac)

	cfg := storage.Config{
		Region: q.region,
	}
	formUploader := storage.NewFormUploader(&cfg)
	ret := storage.PutRet{}

	err = formUploader.Put(ctx, &ret, upToken, key, tmpFile, size, nil)
	if err != nil {
		return 0, fmt.Errorf("Qiniu Put: %w", err)
	}
	return size, nil
}

func (q *QiniuBackupStore) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	deadline := time.Now().Add(1 * time.Hour).Unix()
	privateURL := storage.MakePrivateURL(q.mac, q.domain, key, deadline)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, privateURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create download request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Qiniu download: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("Qiniu download failed: HTTP %d", resp.StatusCode)
	}
	return resp.Body, nil
}

func (q *QiniuBackupStore) Delete(_ context.Context, key string) error {
	err := q.bucketMgr.Delete(q.bucket, key)
	if err != nil {
		return fmt.Errorf("Qiniu Delete: %w", err)
	}
	return nil
}

func (q *QiniuBackupStore) PresignURL(_ context.Context, key string, expiry time.Duration) (string, error) {
	deadline := time.Now().Add(expiry).Unix()
	privateURL := storage.MakePrivateURL(q.mac, q.domain, key, deadline)
	return privateURL, nil
}

func (q *QiniuBackupStore) HeadBucket(_ context.Context) error {
	_, err := q.bucketMgr.GetBucketInfo(q.bucket)
	if err != nil {
		return fmt.Errorf("Qiniu GetBucketInfo failed: %w", err)
	}
	return nil
}

// parseQiniuRegion 将七牛区域标识转为 SDK Region 对象
func parseQiniuRegion(regionID string) *storage.Region {
	switch regionID {
	case "z0":
		return &storage.ZoneHuadong
	case "z1":
		return &storage.ZoneHuabei
	case "z2":
		return &storage.ZoneHuanan
	case "cn-east-2":
		return &storage.ZoneHuadongZheJiang2
	case "na0":
		return &storage.ZoneBeimei
	case "as0":
		return &storage.ZoneXinjiapo
	default:
		return &storage.ZoneHuadong // 默认华东
	}
}
