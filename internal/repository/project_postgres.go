package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/futig/agent-backend/internal/entity"
	"github.com/futig/agent-backend/internal/repository/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ProjectRepository defines the interface for project persistence
type ProjectRepository interface {
	Create(ctx context.Context, project entity.Project) (*entity.Project, error)
	Get(ctx context.Context, id string) (*entity.Project, error)
	List(ctx context.Context, skip, limit int) ([]*entity.Project, error)
	Delete(ctx context.Context, id string) error
}

var _ ProjectRepository = &ProjectPostgres{}

// ProjectPostgres implements ProjectRepository using PostgreSQL with sqlc
type ProjectPostgres struct {
	db      *pgxpool.Pool
	queries *sqlc.Queries
}

func NewProjectPostgres(db *pgxpool.Pool) *ProjectPostgres {
	return &ProjectPostgres{
		db:      db,
		queries: sqlc.New(db),
	}
}

func (r *ProjectPostgres) Create(ctx context.Context, project entity.Project) (*entity.Project, error) {
	projectID, err := uuid.Parse(project.ID)
	if err != nil {
		return nil, fmt.Errorf("parse project ID: %w", err)
	}

	result, err := r.queries.CreateProject(ctx, sqlc.CreateProjectParams{
		ID:          pgtype.UUID{Bytes: projectID, Valid: true},
		Title:       project.Title,
		Description: pgtype.Text{String: project.Description, Valid: project.Description != ""},
	})

	if err != nil {
		return nil, fmt.Errorf("create project: %w", err)
	}

	return toEntityProject(&result), nil
}

func (r *ProjectPostgres) Get(ctx context.Context, id string) (*entity.Project, error) {
	projectID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("parse project ID: %w", err)
	}

	result, err := r.queries.GetProject(ctx, pgtype.UUID{Bytes: projectID, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, entity.ErrProjectNotFound
		}
		return nil, fmt.Errorf("get project: %w", err)
	}

	return toEntityProject(&result), nil
}

func (r *ProjectPostgres) List(ctx context.Context, skip, limit int) ([]*entity.Project, error) {
	results, err := r.queries.ListProjects(ctx, sqlc.ListProjectsParams{
		Limit:  int32(limit),
		Offset: int32(skip),
	})

	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}

	projects := make([]*entity.Project, 0, len(results))
	for _, result := range results {
		projects = append(projects, toEntityProject(&result))
	}

	return projects, nil
}

func (r *ProjectPostgres) Delete(ctx context.Context, id string) error {
	projectID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("parse project ID: %w", err)
	}

	err = r.queries.DeleteProject(ctx, pgtype.UUID{Bytes: projectID, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entity.ErrProjectNotFound
		}
		return fmt.Errorf("delete project: %w", err)
	}

	return nil
}
