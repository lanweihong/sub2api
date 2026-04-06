package service

import (
	"context"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

// PayloadCleanupService 定期清理过期的报文审计记录。
type PayloadCleanupService struct {
	payloadRepo    UsageLogPayloadRepository
	settingService *SettingService
	timingWheel    *TimingWheelService
	startOnce      sync.Once
	stopOnce       sync.Once
}

func NewPayloadCleanupService(
	payloadRepo UsageLogPayloadRepository,
	settingService *SettingService,
	timingWheel *TimingWheelService,
) *PayloadCleanupService {
	return &PayloadCleanupService{
		payloadRepo:    payloadRepo,
		settingService: settingService,
		timingWheel:    timingWheel,
	}
}

// Start 启动每日定期清理。
func (s *PayloadCleanupService) Start() {
	if s == nil || s.payloadRepo == nil || s.timingWheel == nil {
		return
	}
	s.startOnce.Do(func() {
		s.timingWheel.ScheduleRecurring("payload_cleanup", 24*time.Hour, s.runOnce)
		logger.LegacyPrintf("service.payload_cleanup",
			"[PayloadCleanup] started (interval=24h)")
	})
}

// Stop 停止清理任务。
func (s *PayloadCleanupService) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		s.timingWheel.Cancel("payload_cleanup")
	})
}

// runOnce 单次清理执行。
func (s *PayloadCleanupService) runOnce() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	settings, err := s.settingService.GetPayloadLoggingSettings(ctx)
	if err != nil {
		logger.LegacyPrintf("service.payload_cleanup", "[PayloadCleanup] WARN: failed to get settings: %v", err)
		return
	}
	if settings.RetentionDays <= 0 {
		return // 0 = 不自动清理
	}

	cutoff := time.Now().AddDate(0, 0, -settings.RetentionDays)
	deleted, err := s.payloadRepo.DeleteBefore(ctx, cutoff)
	if err != nil {
		logger.LegacyPrintf("service.payload_cleanup", "[PayloadCleanup] WARN: cleanup failed: %v", err)
		return
	}
	if deleted > 0 {
		logger.LegacyPrintf("service.payload_cleanup",
			"[PayloadCleanup] cleanup completed: deleted=%d, cutoff=%s",
			deleted, cutoff.Format(time.RFC3339))
	}
}
