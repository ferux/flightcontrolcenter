package fcontext

import (
	"context"

	"go.uber.org/zap"

	"github.com/ferux/flightcontrolcenter/internal/logger"
	"github.com/ferux/flightcontrolcenter/internal/model"
)

type (
	requestID          struct{}
	loggerKey          struct{}
	DeviceIDKey        struct{}
	zapKey             struct{}
	DeviceRequestIDKey struct{}
)

// WithRequestID adds request id to ctx.
func WithRequestID(ctx context.Context, rid string) context.Context {
	return context.WithValue(ctx, requestID{}, rid)
}

// RequestID gets request id from context or generates a new one.
func RequestID(ctx context.Context) string {
	rid, _ := ctx.Value(requestID{}).(string)
	return rid
}

// WithLogger attaches logger to context.
func WithLogger(ctx context.Context, logger logger.Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, logger)
}

// Logger from context. It may be null so use carefully.
func Logger(ctx context.Context) logger.Logger {
	log, ok := ctx.Value(loggerKey{}).(logger.Logger)
	if !ok {
		return logger.New()
	}
	return log
}

// WithZap adds zap logger to context.
func WithZap(ctx context.Context, z *zap.Logger) context.Context {
	return context.WithValue(ctx, zapKey{}, z)
}

// Zap returns zap logger, stored in context or Nop one.
func Zap(ctx context.Context) *zap.Logger {
	logger, ok := ctx.Value(zapKey{}).(*zap.Logger)
	if !ok {
		return zap.NewNop()
	}

	return logger
}

// WithDeviceID attaches `device_id` to context.
func WithDeviceID(ctx context.Context, deviceID model.DeviceID) context.Context {
	return context.WithValue(ctx, DeviceIDKey{}, deviceID)
}

// DeviceID from context.
func DeviceID(ctx context.Context) model.DeviceID {
	d, _ := ctx.Value(DeviceIDKey{}).(model.DeviceID)
	return d
}

// WithDeviceRequestID is used in keeper package to append request id to context.
func WithDeviceRequestID(ctx context.Context, requestID uint64) context.Context {
	return context.WithValue(ctx, DeviceRequestIDKey{}, requestID)
}

// DeviceRequestID retrieves request id.
func DeviceRequestID(ctx context.Context) uint64 {
	reqID := ctx.Value(DeviceRequestIDKey{}).(uint64)
	return reqID
}
