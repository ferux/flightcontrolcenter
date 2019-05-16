package fcontext

import (
	"context"
)

type requestID struct{}

// WithRequestID adds request id to ctx
func WithRequestID(ctx context.Context, rid string) context.Context {
	return context.WithValue(ctx, requestID{}, rid)
}

// RequestID gets request id from context or generates a new one
func RequestID(ctx context.Context) string {
	rid, _ := ctx.Value(requestID{}).(string)
	return rid
}
