package logger

import (
	"context"

	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
)

// AddFields adds fields to the logger in context and returns new context
func AddFields(ctx context.Context, fields ...zap.Field) context.Context {
	logger := ctxzap.Extract(ctx)
	return ctxzap.ToContext(ctx, logger.With(fields...))
}

// WithAction adds "action" field to context logger to describe the flow
func WithAction(ctx context.Context, action string) context.Context {
	logger := ctxzap.Extract(ctx)
	return ctxzap.ToContext(ctx, logger.With(zap.String("action", action)))
}