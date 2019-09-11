package fcontext

import (
	"context"
	"testing"
)

func TestWithRequestID(t *testing.T) {
	ctx := context.Background()
	ridExp := "test"
	ctx = WithRequestID(ctx, ridExp)

	ridGot, ok := ctx.Value(requestID{}).(string)
	if !ok {
		t.Error("request should be string type")
	}

	if ridGot != ridExp {
		t.Errorf("exp %s got %s", ridExp, ridGot)
	}
}

func TestRequestID(t *testing.T) {
	ridExp := "test"
	ctx := context.WithValue(context.Background(), requestID{}, ridExp)

	ridGot := RequestID(ctx)
	if ridGot != ridExp {
		t.Errorf("exp %s got %s", ridExp, ridGot)
	}
}
