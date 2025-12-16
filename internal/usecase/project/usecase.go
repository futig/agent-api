package project

import (
	"context"
	"fmt"

	"github.com/futig/agent-backend/internal/entity"
	"github.com/futig/agent-backend/internal/pkg/validator"
	"github.com/futig/agent-backend/internal/repository"
	"github.com/google/uuid"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
)

// ProjectUsecase implements project business logic
type ProjectUsecase struct {
	projectRepo     repository.ProjectRepository
	projectFileRepo repository.ProjectFileRepository
	validator       *validator.Validator
	ragConnector    RagConnector
	logger          *zap.Logger
}

// NewUsecase creates a new project use case
func NewUsecase(
	projectRepo repository.ProjectRepository,
	projectFileRepo repository.ProjectFileRepository,
	validator *validator.Validator,
	ragConnector RagConnector,
	logger *zap.Logger,
) *ProjectUsecase {
	return &ProjectUsecase{
		projectRepo:     projectRepo,
		projectFileRepo: projectFileRepo,
		validator:       validator,
		ragConnector:    ragConnector,
		logger:          logger,
	}
}

// CreateProject creates a new project, indexes files in RAG, then saves metadata
func (uc *ProjectUsecase) CreateProject(
	ctx context.Context,
	req *entity.CreateProjectRequest,
) (*entity.Project, error) {
	project := &entity.Project{
		ID:          uuid.New().String(),
		Title:       req.Title,
		Description: req.Description,
	}

	project, err := uc.projectRepo.Create(ctx, *project)
	if err != nil {
		return nil, fmt.Errorf("create project: %w", err)
	}

	ctxzap.Info(ctx, "project created",
		zap.String("project_id", project.ID),
		zap.String("title", req.Title),
	)

	fileDataList, err := uc.prepareFileData(ctx, req.Files)
	if err != nil {
		uc.projectRepo.Delete(ctx, project.ID)
		return nil, fmt.Errorf("prepare files: %w", err)
	}

	if err := uc.ragConnector.IndexFiles(ctx, project.ID, fileDataList); err != nil {
		uc.projectRepo.Delete(ctx, project.ID)
		return nil, fmt.Errorf("index files in RAG: %w", err)
	}

	ctxzap.Info(ctx, "files indexed in RAG successfully", zap.Int("file_count", len(fileDataList)))

	savedFiles, err := uc.saveFileMetadata(ctx, project.ID, req.Files)
	if err != nil {
		uc.ragConnector.DeleteIndex(ctx, project.ID)
		uc.projectRepo.Delete(ctx, project.ID)
		return nil, fmt.Errorf("save file metadata: %w", err)
	}

	project.Files = savedFiles

	ctxzap.Info(ctx, "project created successfully", zap.Int("file_count", len(savedFiles)))

	return project, nil
}

func (uc *ProjectUsecase) AddFiles(ctx context.Context, req *entity.AddFilesRequest) ([]*entity.File, error) {
	if _, err := uc.projectRepo.Get(ctx, req.ProjectID); err != nil {
		return nil, err
	}

	fileDataList, err := uc.prepareFileData(ctx, req.Files)
	if err != nil {
		return nil, fmt.Errorf("prepare files: %w", err)
	}

	if err := uc.ragConnector.IndexFiles(ctx, req.ProjectID, fileDataList); err != nil {
		return nil, fmt.Errorf("index files in RAG: %w", err)
	}

	ctxzap.Info(ctx, "files indexed in RAG successfully", zap.Int("file_count", len(fileDataList)))

	savedFiles, err := uc.saveFileMetadata(ctx, req.ProjectID, req.Files)
	if err != nil {
		return nil, fmt.Errorf("save file metadata: %w", err)
	}

	ctxzap.Info(ctx, "files added successfully", zap.Int("file_count", len(savedFiles)))

	return savedFiles, nil
}

// AddFileFromContent adds a file to an existing project from raw content (non-HTTP context)
// This is used by Telegram bot and other non-multipart contexts
func (uc *ProjectUsecase) AddFileFromContent(
	ctx context.Context,
	projectID string,
	filename string,
	content []byte,
	contentType string,
) (*entity.File, error) {
	// Validate project exists
	if _, err := uc.projectRepo.Get(ctx, projectID); err != nil {
		return nil, err
	}

	// Create FileData for RAG indexing
	fileData := entity.FileData{
		Filename: filename,
		Content:  content,
	}

	// Index in RAG
	if err := uc.ragConnector.IndexFiles(ctx, projectID, []entity.FileData{fileData}); err != nil {
		return nil, fmt.Errorf("index file in RAG: %w", err)
	}

	ctxzap.Info(ctx, "file indexed in RAG successfully",
		zap.String("filename", filename),
		zap.String("project_id", projectID),
	)

	// Save file metadata to database
	fileID := uuid.New().String()
	file := &entity.File{
		ID:          fileID,
		ProjectID:   projectID,
		Filename:    validator.SanitizeFilename(filename),
		Size:        int64(len(content)),
		ContentType: contentType,
	}

	savedFile, err := uc.projectFileRepo.AddFile(ctx, *file)
	if err != nil {
		return nil, fmt.Errorf("save file metadata: %w", err)
	}

	ctxzap.Info(ctx, "file added successfully",
		zap.String("file_id", savedFile.ID),
		zap.String("filename", savedFile.Filename),
	)

	return savedFile, nil
}

// CreateProjectFromContent creates a new project and indexes initial file content (non-HTTP context)
// This is used by Telegram bot to create projects with initial requirements file
func (uc *ProjectUsecase) CreateProjectFromContent(
	ctx context.Context,
	title string,
	description string,
	filename string,
	content []byte,
	contentType string,
) (*entity.Project, error) {
	project := &entity.Project{
		ID:          uuid.New().String(),
		Title:       title,
		Description: description,
	}

	project, err := uc.projectRepo.Create(ctx, *project)
	if err != nil {
		return nil, fmt.Errorf("create project: %w", err)
	}

	ctxzap.Info(ctx, "project created",
		zap.String("project_id", project.ID),
		zap.String("title", title),
	)

	// Index file in RAG
	fileData := entity.FileData{
		Filename: filename,
		Content:  content,
	}

	if err := uc.ragConnector.IndexFiles(ctx, project.ID, []entity.FileData{fileData}); err != nil {
		uc.projectRepo.Delete(ctx, project.ID)
		return nil, fmt.Errorf("index file in RAG: %w", err)
	}

	ctxzap.Info(ctx, "file indexed in RAG successfully",
		zap.String("filename", filename),
		zap.String("project_id", project.ID),
	)

	// Save file metadata to database
	fileID := uuid.New().String()
	file := &entity.File{
		ID:          fileID,
		ProjectID:   project.ID,
		Filename:    validator.SanitizeFilename(filename),
		Size:        int64(len(content)),
		ContentType: contentType,
	}

	savedFile, err := uc.projectFileRepo.AddFile(ctx, *file)
	if err != nil {
		uc.ragConnector.DeleteIndex(ctx, project.ID)
		uc.projectRepo.Delete(ctx, project.ID)
		return nil, fmt.Errorf("save file metadata: %w", err)
	}

	project.Files = []*entity.File{savedFile}

	ctxzap.Info(ctx, "project created successfully with initial file",
		zap.String("project_id", project.ID),
		zap.String("file_id", savedFile.ID),
	)

	return project, nil
}

// ListProjects retrieves projects with pagination
func (uc *ProjectUsecase) ListProjects(ctx context.Context, req *entity.ListProjectsRequest) ([]*entity.Project, error) {
	projects, err := uc.projectRepo.List(ctx, req.Skip, req.Limit)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}

	return projects, nil
}

// GetProject retrieves a project by ID
func (uc *ProjectUsecase) GetProject(ctx context.Context, id string) (*entity.Project, error) {
	if _, err := uuid.Parse(id); err != nil {
		return nil, fmt.Errorf("%w: invalid project ID format", entity.ErrInvalidParameter)
	}

	project, err := uc.projectRepo.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get project: %w", err)
	}

	return project, nil
}

// DeleteProject deletes a project and all its files
func (uc *ProjectUsecase) DeleteProject(ctx context.Context, id string) error {
	if _, err := uuid.Parse(id); err != nil {
		return fmt.Errorf("%w: invalid project ID format", entity.ErrInvalidParameter)
	}

	ctxzap.Info(ctx, "deleting RAG index")
	if err := uc.ragConnector.DeleteIndex(ctx, id); err != nil {
		return fmt.Errorf("delete RAG index: %w", err)
	}

	if err := uc.projectRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete project: %w", err)
	}

	ctxzap.Info(ctx, "project deleted successfully")
	return nil
}

// ListFiles retrieves all files for a project
func (uc *ProjectUsecase) ListFiles(ctx context.Context, projectID string) ([]*entity.File, error) {
	if _, err := uuid.Parse(projectID); err != nil {
		return nil, fmt.Errorf("%w: invalid project ID format", entity.ErrInvalidParameter)
	}

	if _, err := uc.projectRepo.Get(ctx, projectID); err != nil {
		return nil, err
	}

	files, err := uc.projectFileRepo.GetFiles(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list files: %w", err)
	}

	return files, nil
}
