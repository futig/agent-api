package callback

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/futig/agent-backend/internal/config"
	"github.com/futig/agent-backend/internal/entity"
	"github.com/futig/agent-backend/internal/integration/common"
	pkghttp "github.com/futig/agent-backend/pkg/http"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
)

type Connector struct {
	config    config.CallbackConnectorConfig
	connector *pkghttp.Connector
	logger    *zap.Logger
}

func NewConnector(
	cfg config.CallbackConnectorConfig,
	logger *zap.Logger,
) *Connector {
	return &Connector{
		connector: common.NewBaseConnector(cfg.HTTPClientConfig, logger),
		config:    cfg,
		logger:    logger,
	}
}

// SendQuestions sends a questions event to the specified callback URL
func (c *Connector) SendQuestions(ctx context.Context, callbackURL string, requestID string, data *entity.IterationWithQuestions) {
	err := c.Send(ctx, callbackURL, requestID, &entity.CallbackEvent{
		Event: entity.CallbackEventTypeQuestions,
		Data:  data,
	})
	if err != nil {
		ctxzap.Error(ctx, "failed to send questions callback", zap.Error(err))
	}
}

// SendProjectUpdated sends a project updated event to the specified callback URL
func (c *Connector) SendProjectUpdated(ctx context.Context, callbackURL string, requestID string, data *entity.CallbackProjectUpdatedData) {
	err := c.Send(ctx, callbackURL, requestID, &entity.CallbackEvent{
		Event: entity.CallbackEventTypeProjectUpdated,
		Data:  data,
	})
	if err != nil {
		ctxzap.Error(ctx, "failed to send project updated callback", zap.Error(err))
	}
}

// SendFinalResult sends a final result event to the specified callback URL
func (c *Connector) SendFinalResult(ctx context.Context, callbackURL string, requestID string, data *entity.SessionDTO) {
	err := c.Send(ctx, callbackURL, requestID, &entity.CallbackEvent{
		Event: entity.CallbackEventTypeFinalResult,
		Data:  data,
	})
	if err != nil {
		ctxzap.Error(ctx, "failed to send final result callback", zap.Error(err))
	}
}

// SendError sends an error event to the specified callback URL
func (c *Connector) SendError(ctx context.Context, callbackURL string, requestID string, message string, details map[string]any) {
	err := c.Send(ctx, callbackURL, requestID, &entity.CallbackEvent{
		Event: entity.CallbackEventTypeError,
		Data: &entity.CallbackErrorData{
			Error: entity.CallbackErrorDetails{
				Message: message,
				Details: details,
			},
		},
	})
	if err != nil {
		ctxzap.Error(ctx, "failed to send error callback", zap.Error(err))
	}
}

func (c *Connector) Send(ctx context.Context, callbackURL string, requestID string, event *entity.CallbackEvent) error {
	if event.Timestamp == "" {
		event.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}

	ctxzap.Debug(ctx, "sending callback event",
		zap.String("event_type", string(event.Event)),
		zap.String("callback_url", callbackURL),
		zap.String("request_id", requestID),
		zap.String("timestamp", event.Timestamp),
	)

	opts := []pkghttp.RequestOpt{
		pkghttp.WithHeader("X-Request-ID", requestID),
		pkghttp.WithURL(callbackURL),
	}

	err := c.connector.DoRequest(ctx, http.MethodPost, "", event, nil, opts...)
	if err != nil {
		return fmt.Errorf("failed to send callback, event_type: %s, url: %s, error: %w", string(event.Event), callbackURL, err)
	}

	ctxzap.Info(ctx, "callback sent successfully",
		zap.String("event_type", string(event.Event)),
		zap.String("callback_url", callbackURL),
		zap.String("request_id", requestID),
	)
	return nil
}
