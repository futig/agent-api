package repository

import (
	"context"
	"fmt"

	"github.com/futig/agent-backend/internal/entity"
	"github.com/futig/agent-backend/internal/repository/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ProjectFileRepository defines the interface for project file metadata persistence
type ProjectFileRepository interface {
	AddFile(ctx context.Context, file entity.File) (*entity.File, error)
	GetFiles(ctx context.Context, projectID string) ([]*entity.File, error)
	DeleteFile(ctx context.Context, fileID string) error
}

var _ ProjectFileRepository = &ProjectFilePostgres{}

// ProjectFilePostgres implements ProjectFileRepository using PostgreSQL
type ProjectFilePostgres struct {
	db      *pgxpool.Pool
	queries *sqlc.Queries
}

func NewProjectFilePostgres(db *pgxpool.Pool) *ProjectFilePostgres {
	return &ProjectFilePostgres{
		db:      db,
		queries: sqlc.New(db),
	}
}

func (r *ProjectFilePostgres) AddFile(ctx context.Context, file entity.File) (*entity.File, error) {
	fileID, err := uuid.Parse(file.ID)
	if err != nil {
		return nil, fmt.Errorf("parse file ID: %w", err)
	}

	projectID, err := uuid.Parse(file.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("parse project ID: %w", err)
	}

	result, err := r.queries.AddFile(ctx, sqlc.AddFileParams{
		ID:          pgtype.UUID{Bytes: fileID, Valid: true},
		ProjectID:   pgtype.UUID{Bytes: projectID, Valid: true},
		Filename:    file.Filename,
		Size:        file.Size,
		ContentType: file.ContentType,
	})

	if err != nil {
		return nil, fmt.Errorf("add file: %w", err)
	}

	return toEntityFile(&result), nil
}

func (r *ProjectFilePostgres) DeleteFile(ctx context.Context, fileID string) error {
	fid, err := uuid.Parse(fileID)
	if err != nil {
		return fmt.Errorf("parse file ID: %w", err)
	}

	err = r.queries.DeleteProjectFile(ctx, pgtype.UUID{Bytes: fid, Valid: true})
	if err != nil {
		return fmt.Errorf("delete file: %w", err)
	}

	return nil
}

func (r *ProjectFilePostgres) GetFiles(ctx context.Context, projectID string) ([]*entity.File, error) {
	pid, err := uuid.Parse(projectID)
	if err != nil {
		return nil, fmt.Errorf("parse project ID: %w", err)
	}

	results, err := r.queries.GetFiles(ctx, pgtype.UUID{Bytes: pid, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("get files: %w", err)
	}

	files := make([]*entity.File, 0, len(results))
	for _, result := range results {
		files = append(files, toEntityFile(&result))
	}

	return files, nil
}
