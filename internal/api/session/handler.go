package session

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/futig/agent-backend/internal/entity"
	"github.com/futig/agent-backend/internal/pkg/formatter"
	"github.com/futig/agent-backend/internal/pkg/logger"
	"github.com/futig/agent-backend/internal/pkg/validator"
	"github.com/go-chi/chi/v5"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
)

type Handler struct {
	usecase      SessionUsecase
	callbackConn CallbackConnector
	validator    *validator.Validator
}

func NewHandler(
	usecase SessionUsecase,
	validator *validator.Validator,
	callbackConn CallbackConnector,
) *Handler {
	return &Handler{
		usecase:      usecase,
		validator:    validator,
		callbackConn: callbackConn,
	}
}

// StartSession handles POST /interview-session - Start new session
func (h *Handler) StartSession(w http.ResponseWriter, r *http.Request) {
	ctx := logger.WithAction(r.Context(), "StartSession")

	requestID := r.Header.Get("X-Request-ID")

	var req entity.StartSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ctxzap.Error(ctx, "failed to decode request body", zap.Error(err))
		h.respondError(ctx, w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	if err := h.validator.ValidateStartSession(&req); err != nil {
		ctxzap.Error(ctx, "failed to validate request", zap.Error(err))
		h.respondError(ctx, w, http.StatusBadRequest, "validation failed", err)
		return
	}

	ctxzap.Info(ctx, "starting interview session", zap.Any("request", req))

	go func() {
		bgCtx := logger.AddFields(ctxzap.ToContext(context.Background(), ctxzap.Extract(ctx)),
			zap.String("request_id", requestID),
			zap.String("action", "StartSession-async"),
		)

		questionsBlock, err := h.usecase.StartHTTPSession(bgCtx, &req)
		if err != nil {
			ctxzap.Error(bgCtx, "failed to start session", zap.Error(err))
			h.callbackConn.SendError(bgCtx, req.CallbackURL, requestID, "failed to start session", map[string]any{
				"error": err.Error(),
			})
			return
		}

		ctxzap.Info(bgCtx, "session started successfully")

		h.callbackConn.SendQuestions(bgCtx, req.CallbackURL, requestID, questionsBlock)
	}()

	// Return accepted status
	h.respondJSON(w, http.StatusAccepted, map[string]string{
		"status":  "accepted",
		"message": "session creation is being processed",
	})
}

// GetSession handles GET /interview-session/{id} - Get session status
func (h *Handler) GetSession(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sessionID := chi.URLParam(r, "id")

	ctx = logger.AddFields(ctx,
		zap.String("session_id", sessionID),
		zap.String("action", "GetSession"),
	)

	ctxzap.Debug(ctx, "fetching session")

	session, err := h.usecase.GetSession(ctx, sessionID)
	if err != nil {
		h.handleUsecaseError(ctx, w, err)
		return
	}

	ctxzap.Info(ctx, "session fetched successfully",
		zap.Any("session", session),
	)

	h.respondJSON(w, http.StatusOK, toSessionDTO(session))
}

// SubmitTextAnswer handles POST /interview-session/{id}/answers - Submit text answers
func (h *Handler) SubmitTextAnswer(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sessionID := chi.URLParam(r, "id")
	questionID := chi.URLParam(r, "question_id")

	requestID := r.Header.Get("X-Request-ID")

	ctx = logger.AddFields(ctx,
		zap.String("session_id", sessionID),
		zap.String("action", "SubmitTextAnswer"),
	)

	var req entity.SubmitAnswerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ctxzap.Error(ctx, "failed to decode request body", zap.Error(err))
		h.respondError(ctx, w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	if err := h.validator.ValidateSubmitAnswer(&req); err != nil {
		ctxzap.Error(ctx, "failed to validate request", zap.Error(err))
		h.respondError(ctx, w, http.StatusBadRequest, "validation failed", err)
		return
	}

	ctxzap.Info(ctx, "submitting text answer",
		zap.String("question_id", questionID),
		zap.Bool("is_skipped", req.IsSkipped),
	)

	go func() {
		bgCtx := logger.AddFields(ctxzap.ToContext(context.Background(), ctxzap.Extract(ctx)),
			zap.String("request_id", requestID),
			zap.String("session_id", sessionID),
			zap.String("question_id", questionID),
			zap.String("action", "SubmitTextAnswer-async"),
		)

		ctxzap.Info(bgCtx, "processing text answer")

		var iteration *entity.IterationWithQuestions
		var err error

		if req.IsSkipped {
			iteration, err = h.usecase.SkipAnswer(bgCtx, sessionID, questionID)
		} else {
			iteration, err = h.usecase.SubmitTextAnswer(bgCtx, sessionID, questionID, req.Answer)
		}
		if err != nil {
			ctxzap.Error(bgCtx, "failed to submit answer", zap.Error(err))
			h.callbackConn.SendError(bgCtx, req.CallbackURL, requestID, "failed to submit answer", map[string]any{
				"session_id":  sessionID,
				"question_id": questionID,
				"error":       err.Error(),
			})
			return
		}

		if iteration != nil {
			h.callbackConn.SendQuestions(bgCtx, req.CallbackURL, requestID, iteration)
			return
		}

		iteration, err = h.usecase.ValidateAnswers(bgCtx, sessionID)
		if err != nil {
			ctxzap.Error(bgCtx, "failed to validate answers", zap.Error(err))
			h.callbackConn.SendError(bgCtx, req.CallbackURL, requestID, "failed to validate answers", map[string]any{
				"session_id": sessionID,
				"error":      err.Error(),
			})
			return
		}

		if iteration != nil {
			h.callbackConn.SendQuestions(bgCtx, req.CallbackURL, requestID, iteration)
			return
		}

		session, err := h.usecase.GenerateSummary(bgCtx, sessionID)
		if err != nil {
			ctxzap.Error(bgCtx, "failed to generate summary", zap.Error(err))
			h.callbackConn.SendError(bgCtx, req.CallbackURL, requestID, "failed to generate summary", map[string]any{
				"session_id": sessionID,
				"error":      err.Error(),
			})
			return
		}

		h.callbackConn.SendFinalResult(bgCtx, req.CallbackURL, requestID, toSessionDTO(session))
	}()

	h.respondJSON(w, http.StatusAccepted, map[string]string{
		"status":  "accepted",
		"message": "answer is being processed",
	})
}

// SubmitAudioAnswer handles POST /interview-session/{id}/answers/audio - Submit audio answers
func (h *Handler) SubmitAudioAnswer(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sessionID := chi.URLParam(r, "id")
	questionID := chi.URLParam(r, "question_id")

	requestID := r.Header.Get("X-Request-ID")

	ctx = logger.AddFields(ctx,
		zap.String("session_id", sessionID),
		zap.String("action", "SubmitAudioAnswer"),
	)

	// Parse multipart form (max 10MB)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		ctxzap.Error(ctx, "failed to parse multipart form", zap.Error(err))
		h.respondError(ctx, w, http.StatusBadRequest, "failed to parse form", err)
		return
	}

	file, header, err := r.FormFile("audio")
	if err != nil {
		ctxzap.Error(ctx, "missing audio file", zap.Error(err))
		h.respondError(ctx, w, http.StatusBadRequest, "audio file is required", err)
		return
	}
	defer file.Close()

	// Get is_skipped and callback_url from multipart form body
	callbackURL := r.FormValue("callback_url")
	isSkippedStr := r.FormValue("is_skipped")
	isSkipped := isSkippedStr == "true"

	req := entity.SubmitAudioAnswerRequest{
		AudioFile:   header,
		IsSkipped:   isSkipped,
		CallbackURL: callbackURL,
	}

	if err := h.validator.ValidateSubmitAudioAnswer(&req); err != nil {
		ctxzap.Error(ctx, "failed to validate request", zap.Error(err))
		h.respondError(ctx, w, http.StatusBadRequest, "validation failed", err)
		return
	}

	ctxzap.Info(ctx, "submitting audio answer",
		zap.Int64("size_bytes", header.Size),
		zap.Bool("is_skipped", isSkipped),
	)

	go func() {
		bgCtx := logger.AddFields(ctxzap.ToContext(context.Background(), ctxzap.Extract(ctx)),
			zap.String("request_id", requestID),
			zap.String("session_id", sessionID),
			zap.String("question_id", questionID),
			zap.String("action", "SubmitAudioAnswer-async"),
		)

		ctxzap.Info(bgCtx, "processing audio answer")

		var iteration *entity.IterationWithQuestions
		var err error

		if req.IsSkipped {
			iteration, err = h.usecase.SkipAnswer(bgCtx, sessionID, questionID)
		} else {
			iteration, err = h.usecase.SubmitHTTPAudioAnswer(bgCtx, sessionID, questionID, header)
		}
		if err != nil {
			ctxzap.Error(bgCtx, "failed to submit audio answer", zap.Error(err))
			h.callbackConn.SendError(bgCtx, req.CallbackURL, requestID, "failed to process audio answer", map[string]any{
				"session_id":  sessionID,
				"question_id": questionID,
				"error":       err.Error(),
			})
			return
		}

		if iteration != nil {
			h.callbackConn.SendQuestions(bgCtx, req.CallbackURL, requestID, iteration)
			return
		}

		iteration, err = h.usecase.ValidateAnswers(bgCtx, sessionID)
		if err != nil {
			ctxzap.Error(bgCtx, "failed to validate answers", zap.Error(err))
			h.callbackConn.SendError(bgCtx, req.CallbackURL, requestID, "failed to validate answers", map[string]any{
				"session_id": sessionID,
				"error":      err.Error(),
			})
			return
		}

		if iteration != nil {
			h.callbackConn.SendQuestions(bgCtx, req.CallbackURL, requestID, iteration)
			return
		}

		session, err := h.usecase.GenerateSummary(bgCtx, sessionID)
		if err != nil {
			ctxzap.Error(bgCtx, "failed to generate summary", zap.Error(err))
			h.callbackConn.SendError(bgCtx, req.CallbackURL, requestID, "failed to generate summary", map[string]any{
				"session_id": sessionID,
				"error":      err.Error(),
			})
			return
		}

		h.callbackConn.SendFinalResult(bgCtx, req.CallbackURL, requestID, toSessionDTO(session))
	}()

	h.respondJSON(w, http.StatusAccepted, map[string]string{
		"status":  "accepted",
		"message": "audio answer is being processed",
	})
}

// GetSessionResult handles GET /interview-session/{id}/result - Get final result
func (h *Handler) GetSessionResult(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sessionID := chi.URLParam(r, "id")

	ctx = logger.AddFields(ctx,
		zap.String("session_id", sessionID),
		zap.String("action", "GetSessionResult"),
	)

	formatParam := r.URL.Query().Get("format")
	if formatParam == "" {
		formatParam = "markdown"
	}

	format := entity.ResultFormat(formatParam)
	if !format.IsValid() {
		ctxzap.Warn(ctx, "invalid format parameter", zap.String("format", formatParam))
		h.respondError(ctx, w, http.StatusBadRequest, "invalid format parameter",
			fmt.Errorf("format must be one of: markdown, json, docx, pdf"))
		return
	}

	ctx = logger.AddFields(ctx, zap.String("format", string(format)))
	ctxzap.Debug(ctx, "fetching session result")

	result, err := h.usecase.GetSessionResult(ctx, sessionID)
	if err != nil {
		h.handleUsecaseError(ctx, w, err)
		return
	}

	// Create formatter
	factory := formatter.NewFactory()
	fmtr, err := factory.Create(format)
	if err != nil {
		ctxzap.Error(ctx, "format not implemented", zap.Error(err))
		h.respondError(ctx, w, http.StatusNotImplemented, "format not implemented", err)
		return
	}

	formattedResult, err := fmtr.Format(result)
	if err != nil {
		ctxzap.Error(ctx, "failed to format result", zap.Error(err))
		h.respondError(ctx, w, http.StatusInternalServerError, "failed to format result", err)
		return
	}

	ctxzap.Info(ctx, "session result fetched and formatted successfully")
	w.Header().Set("Content-Type", fmtr.ContentType())
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"requirements-%s%s\"", sessionID, fmtr.FileExtension()))
	w.WriteHeader(http.StatusOK)
	w.Write(formattedResult)
}

// CancelSession handles POST /interview-session/{id}/cancel - Cancel session
func (h *Handler) CancelSession(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sessionID := chi.URLParam(r, "id")

	ctx = logger.AddFields(ctx,
		zap.String("session_id", sessionID),
		zap.String("action", "CancelSession"),
	)

	ctxzap.Info(ctx, "cancelling session")

	if err := h.usecase.CancelSession(ctx, sessionID); err != nil {
		h.handleUsecaseError(ctx, w, err)
		return
	}

	ctxzap.Info(ctx, "session cancelled successfully")
	h.respondJSON(w, http.StatusOK, map[string]string{
		"message": "session cancelled successfully",
	})
}

// Helper methods
func (h *Handler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *Handler) respondError(ctx context.Context, w http.ResponseWriter, status int, message string, err error) {
	ctxzap.Error(ctx, message, zap.Error(err))
	h.respondJSON(w, status, entity.ErrorResponse{
		Error:   http.StatusText(status),
		Message: message,
	})
}

func (h *Handler) handleUsecaseError(ctx context.Context, w http.ResponseWriter, err error) {
	if errors.Is(err, entity.ErrSessionNotFound) || errors.Is(err, entity.ErrProjectNotFound) || errors.Is(err, entity.ErrIterationNotFound) {
		h.respondError(ctx, w, http.StatusNotFound, "resource not found", err)
	} else if errors.Is(err, entity.ErrInvalidParameter) || errors.Is(err, entity.ErrInvalidFormat) || errors.Is(err, entity.ErrMissingField) {
		h.respondError(ctx, w, http.StatusBadRequest, "invalid parameter", err)
	} else if errors.Is(err, entity.ErrSessionNotActive) || errors.Is(err, entity.ErrSessionCancelled) || errors.Is(err, entity.ErrSessionCompleted) || errors.Is(err, entity.ErrInvalidSessionStatus) || errors.Is(err, entity.ErrNoResult) {
		h.respondError(ctx, w, http.StatusConflict, "invalid session state", err)
	} else if errors.Is(err, entity.ErrInvalidExtension) || errors.Is(err, entity.ErrFileTooLarge) {
		h.respondError(ctx, w, http.StatusBadRequest, "invalid file", err)
	} else {
		h.respondError(ctx, w, http.StatusInternalServerError, "internal server error", err)
	}
}
