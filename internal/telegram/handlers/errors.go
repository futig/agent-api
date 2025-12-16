package handlers

import (
	"context"
	"errors"
	"net"

	"github.com/futig/agent-backend/internal/entity"
	"github.com/futig/agent-backend/internal/telegram/render"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
)

// ErrorSeverity represents the severity level of an error
type ErrorSeverity int

const (
	SeverityWarning ErrorSeverity = iota
	SeverityError
	SeverityCritical
)

// String returns string representation of error severity
func (s ErrorSeverity) String() string {
	switch s {
	case SeverityWarning:
		return "warning"
	case SeverityError:
		return "error"
	case SeverityCritical:
		return "critical"
	default:
		return "unknown"
	}
}

// HandlerError represents a structured error with user message and logging info
type HandlerError struct {
	Err         error
	UserMessage string
	LogMessage  string
	Severity    ErrorSeverity
}

// classifyHandlerError analyzes an error and returns a HandlerError with appropriate severity and messages
func classifyHandlerError(err error) *HandlerError {
	if err == nil {
		return &HandlerError{
			Err:         nil,
			UserMessage: render.ErrGeneric,
			LogMessage:  "unknown error",
			Severity:    SeverityWarning,
		}
	}

	// Check for domain errors (non-critical)
	switch {
	case errors.Is(err, entity.ErrProjectNotFound):
		return &HandlerError{
			Err:         err,
			UserMessage: "❌ Проект не найден",
			LogMessage:  "project not found",
			Severity:    SeverityWarning,
		}
	case errors.Is(err, entity.ErrSessionNotFound):
		return &HandlerError{
			Err:         err,
			UserMessage: "❌ Сессия не найдена. Нажмите /start",
			LogMessage:  "session not found",
			Severity:    SeverityWarning,
		}
	case errors.Is(err, entity.ErrQuestionNotFound):
		return &HandlerError{
			Err:         err,
			UserMessage: "❌ Вопрос не найден",
			LogMessage:  "question not found",
			Severity:    SeverityWarning,
		}
	case errors.Is(err, entity.ErrSessionNotActive):
		return &HandlerError{
			Err:         err,
			UserMessage: "❌ Сессия неактивна. Нажмите /start",
			LogMessage:  "session not active",
			Severity:    SeverityWarning,
		}
	}

	// Check for timeout errors
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return &HandlerError{
			Err:         err,
			UserMessage: render.ErrTimeout,
			LogMessage:  "operation timed out",
			Severity:    SeverityError,
		}
	}

	// Check for network errors
	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return &HandlerError{
				Err:         err,
				UserMessage: render.ErrTimeout,
				LogMessage:  "network timeout",
				Severity:    SeverityError,
			}
		}
		return &HandlerError{
			Err:         err,
			UserMessage: render.ErrNetworkIssue,
			LogMessage:  "network error",
			Severity:    SeverityError,
		}
	}

	// Default to generic error
	return &HandlerError{
		Err:         err,
		UserMessage: render.ErrGeneric,
		LogMessage:  "handler error",
		Severity:    SeverityError,
	}
}

// HandleError provides centralized error handling for all handlers
// It logs the error with appropriate severity and sends a user-friendly message
func (h *BaseHandler) HandleError(ctx context.Context, chatID int64, err error) {
	if err == nil {
		return
	}

	handlerErr := classifyHandlerError(err)

	// Log with appropriate severity level
	switch handlerErr.Severity {
	case SeverityCritical:
		ctxzap.Error(ctx, handlerErr.LogMessage,
			zap.Error(handlerErr.Err),
			zap.Int64("chat_id", chatID),
		)
	case SeverityError:
		ctxzap.Error(ctx, handlerErr.LogMessage,
			zap.Error(handlerErr.Err),
			zap.Int64("chat_id", chatID),
		)
	case SeverityWarning:
		ctxzap.Warn(ctx, handlerErr.LogMessage,
			zap.Error(handlerErr.Err),
			zap.Int64("chat_id", chatID),
		)
	}

	// Send user-friendly message
	if h.messageSender != nil {
		h.messageSender.Send(chatID, handlerErr.UserMessage, nil)
	}
}
