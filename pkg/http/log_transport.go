package http

import (
	"net/http"

	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
)

// context keys for attaching request metadata
type payloadContextKey struct{}
type bodySizeContextKey struct{}

type logTransport struct {
	transport http.RoundTripper
}

func (t *logTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := req.Context()

	fields := []zap.Field{
		zap.String("method", req.Method),
		zap.String("url", req.URL.String()),
		zap.Any("headers", req.Header),
	}

	if payload, ok := ctx.Value(payloadContextKey{}).([]byte); ok && len(payload) > 0 {
		fields = append(fields, zap.ByteString("payload", payload))
	}

	ctxzap.Debug(ctx, "HTTP outbound request", fields...)

	return t.transport.RoundTrip(req)
}

// WithRequestLogging wraps the HTTP transport with logging of method, URL, headers and payload metadata.
func WithRequestLogging() HttpOpts {
	return WithTransport(func(rt http.RoundTripper) http.RoundTripper {
		return &logTransport{
			transport: rt,
		}
	})
}

