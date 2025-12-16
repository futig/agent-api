package project

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/futig/agent-backend/internal/config"
	"github.com/futig/agent-backend/internal/entity"
	"github.com/futig/agent-backend/internal/pkg/logger"
	"github.com/futig/agent-backend/internal/pkg/validator"
	"github.com/go-chi/chi/v5"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
)

type Handler struct {
	usecase      ProjectUsecase
	cfg          config.FileUploadConfig
	callbackConn CallbackConnector
	validator    *validator.Validator
}

func NewHandler(
	usecase ProjectUsecase,
	cfg config.FileUploadConfig,
	callbackConn CallbackConnector,
	validator *validator.Validator,
) *Handler {
	return &Handler{
		usecase:      usecase,
		cfg:          cfg,
		callbackConn: callbackConn,
		validator:    validator,
	}
}

// CreateProject handles POST /projects
func (h *Handler) CreateProject(w http.ResponseWriter, r *http.Request) {
	ctx := logger.WithAction(r.Context(), "CreateProject")

	requestID := r.Header.Get("X-Request-ID")

	if err := r.ParseMultipartForm(h.cfg.MaxUploadSize); err != nil {
		ctxzap.Error(ctx, "failed to parse multipart form", zap.Error(err))
		h.respondError(ctx, w, http.StatusBadRequest, "invalid form data or size too large", err)
		return
	}

	req := entity.CreateProjectRequest{
		Title:       r.FormValue("title"),
		Description: r.FormValue("description"),
		CallbackURL: r.FormValue("callback_url"),
	}

	req.Files = r.MultipartForm.File["files"]
	if len(req.Files) == 0 {
		ctxzap.Warn(ctx, "no files provided")
		h.respondError(ctx, w, http.StatusBadRequest, "at least one file is required", nil)
		return
	}

	if err := h.validator.ValidateCreateProject(&req); err != nil {
		ctxzap.Error(ctx, "failed to validate request", zap.Error(err))
		h.respondError(ctx, w, http.StatusBadRequest, "validation failed", err)
		return
	}

	ctxzap.Info(ctx, "creating project",
		zap.String("title", req.Title),
		zap.String("description", req.Description),
		zap.Int("file_count", len(req.Files)),
	)

	// Return accepted status
	h.respondJSON(w, http.StatusAccepted, map[string]string{
		"status":  "accepted",
		"message": "project creation is being processed",
	})

	// Process creation and indexing asynchronously
	go func() {
		bgCtx := logger.AddFields(ctxzap.ToContext(context.Background(), ctxzap.Extract(ctx)),
			zap.String("request_id", requestID),
			zap.String("action", "CreateProject-async"),
		)

		proj, err := h.usecase.CreateProject(bgCtx, &req)
		if err != nil {
			ctxzap.Error(bgCtx, "failed to create project", zap.Error(err))
			h.callbackConn.SendError(bgCtx, req.CallbackURL, requestID, "failed to create project", map[string]any{
				"error": err.Error(),
			})
			return
		}

		ctxzap.Info(bgCtx, "project created successfully", zap.String("project_id", proj.ID))

		h.callbackConn.SendProjectUpdated(bgCtx, req.CallbackURL, requestID, toCallbackProjectUpdated(proj))
	}()
}

// ListProjects handles GET /projects
func (h *Handler) ListProjects(w http.ResponseWriter, r *http.Request) {
	ctx := logger.WithAction(r.Context(), "ListProjects")

	skip, _ := strconv.Atoi(r.URL.Query().Get("skip"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	req := entity.ListProjectsRequest{
		Skip:  skip,
		Limit: limit,
	}

	req.Normalize()

	ctxzap.Debug(ctx, "listing projects",
		zap.Int("skip", skip),
		zap.Int("limit", limit),
	)

	projects, err := h.usecase.ListProjects(ctx, &req)
	if err != nil {
		h.handleUsecaseError(ctx, w, err)
		return
	}

	summaries := make([]*entity.ProjectSummary, 0, len(projects))
	for _, p := range projects {
		summaries = append(summaries, toProjectSummary(p))
	}

	ctxzap.Info(ctx, "projects listed successfully", zap.Int("count", len(summaries)))

	h.respondJSON(w, http.StatusOK, &entity.ListProjectsResponse{
		Projects: summaries,
	})
}

// GetProject handles GET /projects/{project_id}
func (h *Handler) GetProject(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectID := chi.URLParam(r, "project_id")

	ctx = logger.AddFields(ctx,
		zap.String("project_id", projectID),
		zap.String("action", "GetProject"),
	)

	ctxzap.Debug(ctx, "fetching project")

	proj, err := h.usecase.GetProject(ctx, projectID)
	if err != nil {
		h.handleUsecaseError(ctx, w, err)
		return
	}

	ctxzap.Info(ctx, "project fetched successfully")
	h.respondJSON(w, http.StatusOK, toProjectDetail(proj))
}

// DeleteProject handles DELETE /projects/{project_id}
func (h *Handler) DeleteProject(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectID := chi.URLParam(r, "project_id")

	ctx = logger.AddFields(ctx,
		zap.String("project_id", projectID),
		zap.String("action", "DeleteProject"),
	)

	ctxzap.Info(ctx, "deleting project")

	if err := h.usecase.DeleteProject(ctx, projectID); err != nil {
		h.handleUsecaseError(ctx, w, err)
		return
	}

	ctxzap.Info(ctx, "project deleted successfully")
	h.respondJSON(w, http.StatusOK, &entity.DeleteProjectResponse{
		Status: "deleted",
	})
}

// AddFiles handles POST /projects/{project_id}
func (h *Handler) AddFiles(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectID := chi.URLParam(r, "project_id")

	requestID := r.Header.Get("X-Request-ID")

	ctx = logger.AddFields(ctx,
		zap.String("project_id", projectID),
		zap.String("action", "AddFiles"),
	)

	if err := r.ParseMultipartForm(h.cfg.MaxUploadSize); err != nil {
		ctxzap.Error(ctx, "failed to parse multipart form", zap.Error(err))
		h.respondError(ctx, w, http.StatusBadRequest, "invalid form data or size too large", err)
		return
	}

	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		ctxzap.Warn(ctx, "no files provided")
		h.respondError(ctx, w, http.StatusBadRequest, "at least one file is required", nil)
		return
	}

	req := entity.AddFilesRequest{
		Files:       files,
		ProjectID:   projectID,
		CallbackURL: r.FormValue("callback_url"),
	}

	ctxzap.Info(ctx, "adding files to project", zap.Int("file_count", len(files)))

	// Return accepted status
	h.respondJSON(w, http.StatusAccepted, map[string]string{
		"status":  "accepted",
		"message": "files are being processed",
	})

	// Process file addition and indexing asynchronously
	go func() {
		bgCtx := logger.AddFields(ctxzap.ToContext(context.Background(), ctxzap.Extract(ctx)),
			zap.String("request_id", requestID),
			zap.String("project_id", projectID),
			zap.String("action", "AddFiles-async"),
		)

		savedFiles, err := h.usecase.AddFiles(bgCtx, &req)
		if err != nil {
			ctxzap.Error(bgCtx, "failed to add files", zap.Error(err))
			h.callbackConn.SendError(bgCtx, req.CallbackURL, requestID, "failed to add files", map[string]any{
				"project_id": projectID,
				"error":      err.Error(),
				"file_count": len(files),
			})
			return
		}

		ctxzap.Info(bgCtx, "files added successfully", zap.Int("file_count", len(savedFiles)))

		// Send success callback with up-to-date file list
		proj, err := h.usecase.GetProject(bgCtx, projectID)
		if err != nil {
			ctxzap.Warn(bgCtx, "failed to get project for callback", zap.Error(err))
			return
		}

		files, err := h.usecase.ListFiles(bgCtx, projectID)
		if err != nil {
			ctxzap.Warn(bgCtx, "failed to list project files for callback", zap.Error(err))
		} else {
			proj.Files = files
		}

		h.callbackConn.SendProjectUpdated(bgCtx, req.CallbackURL, requestID, toCallbackProjectUpdated(proj))
	}()
}

// ListFiles handles GET /projects/{project_id}/files
func (h *Handler) ListFiles(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectID := chi.URLParam(r, "project_id")

	ctx = logger.AddFields(ctx,
		zap.String("project_id", projectID),
		zap.String("action", "ListFiles"),
	)

	ctxzap.Debug(ctx, "listing files")

	files, err := h.usecase.ListFiles(ctx, projectID)
	if err != nil {
		h.handleUsecaseError(ctx, w, err)
		return
	}

	// Convert to response format
	fileDetails := make([]*entity.FileDetail, 0, len(files))
	for _, f := range files {
		fileDetails = append(fileDetails, toFileDetail(f))
	}

	ctxzap.Info(ctx, "files listed successfully", zap.Int("count", len(fileDetails)))
	h.respondJSON(w, http.StatusOK, &entity.ListFilesResponse{
		Files: fileDetails,
	})
}

// Helper methods
func (h *Handler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *Handler) respondError(ctx context.Context, w http.ResponseWriter, status int, message string, err error) {
	if err != nil {
		ctxzap.Error(ctx, message, zap.Error(err))
	} else {
		ctxzap.Error(ctx, message)
	}
	h.respondJSON(w, status, entity.ErrorResponse{
		Error:   http.StatusText(status),
		Message: message,
	})
}

func (h *Handler) handleUsecaseError(ctx context.Context, w http.ResponseWriter, err error) {
	if errors.Is(err, entity.ErrProjectNotFound) {
		h.respondError(ctx, w, http.StatusNotFound, "resource not found", err)
	} else if errors.Is(err, entity.ErrInvalidParameter) || errors.Is(err, entity.ErrMissingField) {
		h.respondError(ctx, w, http.StatusBadRequest, "invalid parameter", err)
	} else if errors.Is(err, entity.ErrInvalidFile) || errors.Is(err, entity.ErrFileTooLarge) || errors.Is(err, entity.ErrTooManyFiles) || errors.Is(err, entity.ErrInvalidExtension) || errors.Is(err, entity.ErrTotalSizeTooLarge) {
		h.respondError(ctx, w, http.StatusBadRequest, "invalid file", err)
	} else {
		h.respondError(ctx, w, http.StatusInternalServerError, "internal server error", err)
	}
}
