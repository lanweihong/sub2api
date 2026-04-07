package repository

import (
	"context"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

// NewBackupStoreFactory returns a BackupObjectStoreFactory that creates
// the appropriate store based on the Provider field in BackupStorageConfig.
func NewBackupStoreFactory() service.BackupObjectStoreFactory {
	return func(ctx context.Context, cfg *service.BackupStorageConfig) (service.BackupObjectStore, error) {
		cfg.NormalizeProvider()
		switch cfg.Provider {
		case service.BackupStorageProviderS3:
			return NewS3BackupStore(ctx, cfg)
		case service.BackupStorageProviderOSS:
			return NewOSSBackupStore(ctx, cfg)
		case service.BackupStorageProviderQiniu:
			return NewQiniuBackupStore(ctx, cfg)
		default:
			return nil, fmt.Errorf("unsupported backup storage provider: %s", cfg.Provider)
		}
	}
}
