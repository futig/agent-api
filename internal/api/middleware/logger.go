package middleware

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
)

// Logger is a middleware that logs HTTP requests
func Logger(logger *zap.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			logger.Info("Start handle HTTP request",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.String("remote_addr", r.RemoteAddr),
				zap.String("request_id", middleware.GetReqID(r.Context())),
			)

			requestID := middleware.GetReqID(r.Context())
			reqLogger := logger.With(zap.String("request_id", requestID))
			ctx := ctxzap.ToContext(r.Context(), reqLogger)
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r.WithContext(ctx))

			logger.Info("Finish handle HTTP request",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.Int("status", ww.Status()),
				zap.Int("bytes", ww.BytesWritten()),
				zap.Int64("duration_ms", time.Since(start).Milliseconds()),
				zap.String("remote_addr", r.RemoteAddr),
				zap.String("request_id", middleware.GetReqID(r.Context())),
			)
		})
	}
}
