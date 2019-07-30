package fcontext

import (
	"context"

	"github.com/ferux/flightcontrolcenter/internal/logger"
	"github.com/ferux/flightcontrolcenter/internal/model"
)

type requestID struct{}
type loggerKey struct{}
type DeviceIDKey struct{}

// WithRequestID adds request id to ctx
func WithRequestID(ctx context.Context, rid string) context.Context {
	return context.WithValue(ctx, requestID{}, rid)
}

// RequestID gets request id from context or generates a new one
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
	logger, _ := ctx.Value(loggerKey{}).(logger.Logger)
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
