package handler

import (
	"context"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"go.uber.org/zap"
)

func payloadAuditHandlerLogger(base *zap.Logger) *zap.Logger {
	if base != nil {
		return base
	}
	return logger.L()
}

func payloadAuditHandlerContextFields(ctx context.Context) []zap.Field {
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

func logPayloadAuditCaptureDecision(
	base *zap.Logger,
	ctx context.Context,
	route string,
	requestPayloadHash string,
	enabled bool,
	maxRequestSize int64,
	maxResponseSize int64,
	requestBodyLen int,
	requestPayloadLen int,
	responsePayloadLen int,
	requestTruncated bool,
	responseTruncated bool,
	hasResponseBody bool,
) {
	fields := append(payloadAuditHandlerContextFields(ctx),
		zap.String("route", route),
		zap.String("request_payload_hash", strings.TrimSpace(requestPayloadHash)),
		zap.Bool("payload_logging_enabled", enabled),
		zap.Int64("config_max_request_size", maxRequestSize),
		zap.Int64("config_max_response_size", maxResponseSize),
		zap.Int("request_body_len", requestBodyLen),
		zap.Int("captured_request_payload_len", requestPayloadLen),
		zap.Int("captured_response_payload_len", responsePayloadLen),
		zap.Bool("request_truncated", requestTruncated),
		zap.Bool("response_truncated", responseTruncated),
		zap.Bool("has_response_body", hasResponseBody),
	)
	payloadAuditHandlerLogger(base).Info("payload_audit.capture_decision", fields...)
}

func logPayloadAuditRecordTask(
	base *zap.Logger,
	ctx context.Context,
	route string,
	requestPayloadHash string,
	requestPayloadLen int,
	responsePayloadLen int,
	requestTruncated bool,
	responseTruncated bool,
) {
	fields := append(payloadAuditHandlerContextFields(ctx),
		zap.String("route", route),
		zap.String("request_payload_hash", strings.TrimSpace(requestPayloadHash)),
		zap.Int("captured_request_payload_len", requestPayloadLen),
		zap.Int("captured_response_payload_len", responsePayloadLen),
		zap.Bool("request_truncated", requestTruncated),
		zap.Bool("response_truncated", responseTruncated),
	)
	payloadAuditHandlerLogger(base).Info("payload_audit.record_usage_task", fields...)
}
