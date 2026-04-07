package service

import (
	"context"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"go.uber.org/zap"
)

func payloadAuditServiceContextFields(ctx context.Context) []zap.Field {
	fields := make([]zap.Field, 0, 2)
	if ctx == nil {
		return fields
	}
	if requestID, _ := ctx.Value(ctxkey.RequestID).(string); strings.TrimSpace(requestID) != "" {
		fields = append(fields, zap.String("request_id", strings.TrimSpace(requestID)))
	}
	if clientRequestID, _ := ctx.Value(ctxkey.ClientRequestID).(string); strings.TrimSpace(clientRequestID) != "" {
		fields = append(fields, zap.String("client_request_id", strings.TrimSpace(clientRequestID)))
	}
	return fields
}

func payloadAuditServiceLogger(component string, ctx context.Context) *zap.Logger {
	fields := payloadAuditServiceContextFields(ctx)
	if strings.TrimSpace(component) != "" {
		fields = append(fields, zap.String("component", strings.TrimSpace(component)))
	}
	return logger.L().With(fields...)
}

func logPayloadAuditPersistenceSkipped(
	ctx context.Context,
	component string,
	usageRequestID string,
	skipReason string,
	usageLogID int64,
	requestPayloadLen int,
	responsePayloadLen int,
	requestTruncated bool,
	responseTruncated bool,
) {
	payloadAuditServiceLogger(component, ctx).Info("payload_audit.persistence_skipped",
		zap.String("usage_request_id", strings.TrimSpace(usageRequestID)),
		zap.Int64("usage_log_id", usageLogID),
		zap.Int("request_payload_len", requestPayloadLen),
		zap.Int("response_payload_len", responsePayloadLen),
		zap.Bool("request_truncated", requestTruncated),
		zap.Bool("response_truncated", responseTruncated),
		zap.String("skip_reason", skipReason),
	)
}

func logPayloadAuditPersisted(
	ctx context.Context,
	component string,
	usageRequestID string,
	usageLogID int64,
	requestPayloadLen int,
	responsePayloadLen int,
	requestTruncated bool,
	responseTruncated bool,
) {
	payloadAuditServiceLogger(component, ctx).Info("payload_audit.persisted",
		zap.String("usage_request_id", strings.TrimSpace(usageRequestID)),
		zap.Int64("usage_log_id", usageLogID),
		zap.Int("request_payload_len", requestPayloadLen),
		zap.Int("response_payload_len", responsePayloadLen),
		zap.Bool("request_truncated", requestTruncated),
		zap.Bool("response_truncated", responseTruncated),
	)
}

func persistPayloadAuditIfNeeded(
	ctx context.Context,
	component string,
	usageRequestID string,
	repo UsageLogPayloadRepository,
	usageLogID int64,
	reqPayload []byte,
	respPayload []byte,
	reqTruncated bool,
	respTruncated bool,
) {
	reqLen := len(reqPayload)
	respLen := len(respPayload)
	if usageLogID <= 0 {
		logPayloadAuditPersistenceSkipped(
			ctx,
			component,
			usageRequestID,
			"no_usage_log_id",
			usageLogID,
			reqLen,
			respLen,
			reqTruncated,
			respTruncated,
		)
		return
	}
	if reqLen == 0 && respLen == 0 {
		logPayloadAuditPersistenceSkipped(
			ctx,
			component,
			usageRequestID,
			"empty_request_and_response_payload",
			usageLogID,
			reqLen,
			respLen,
			reqTruncated,
			respTruncated,
		)
		return
	}
	writePayloadBestEffort(
		ctx,
		component,
		usageRequestID,
		repo,
		usageLogID,
		reqPayload,
		respPayload,
		reqTruncated,
		respTruncated,
	)
}
