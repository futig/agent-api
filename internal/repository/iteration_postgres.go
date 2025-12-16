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

// IterationRepository defines the interface for iteration persistence
type IterationRepository interface {
	CreateIteration(ctx context.Context, iteration entity.Iteration) (*entity.Iteration, error)
	GetIterationByID(ctx context.Context, id string) (*entity.Iteration, error)
	GetNextIteration(ctx context.Context, sessionID string) (*entity.Iteration, error)
	GetCurrentIteration(ctx context.Context, sessionID string) (*entity.Iteration, error)
	ListIterationsBySession(ctx context.Context, sessionID string) ([]*entity.Iteration, error)
	GetMaxIterationNumber(ctx context.Context, sessionID string) (int, error)
}

var _ IterationRepository = &IterationPostgres{}

// IterationPostgres implements IterationRepository using PostgreSQL
type IterationPostgres struct {
	db      *pgxpool.Pool
	queries *sqlc.Queries
}

func NewIterationPostgres(db *pgxpool.Pool) *IterationPostgres {
	return &IterationPostgres{
		db:      db,
		queries: sqlc.New(db),
	}
}

func (r *IterationPostgres) CreateIteration(ctx context.Context, iteration entity.Iteration) (*entity.Iteration, error) {
	iterID, err := uuid.Parse(iteration.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid iteration ID: %w", err)
	}

	sessionID, err := uuid.Parse(iteration.SessionID)
	if err != nil {
		return nil, fmt.Errorf("invalid session ID: %w", err)
	}

	params := sqlc.CreateIterationParams{
		ID: pgtype.UUID{
			Bytes: iterID,
			Valid: true,
		},
		SessionID: pgtype.UUID{
			Bytes: sessionID,
			Valid: true,
		},
		IterationNumber: int32(iteration.IterationNumber),
		Title:           iteration.Title,
	}

	dbIter, err := r.queries.CreateIteration(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("create iteration: %w", err)
	}

	return toEntityIteration(&dbIter), nil
}

func (r *IterationPostgres) GetIterationByID(ctx context.Context, id string) (*entity.Iteration, error) {
	iterID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid iteration ID: %w", err)
	}

	dbIter, err := r.queries.GetIterationByID(ctx, pgtype.UUID{
		Bytes: iterID,
		Valid: true,
	})
	if err != nil {
		return nil, fmt.Errorf("get iteration: %w", err)
	}

	return toEntityIteration(&dbIter), nil
}

func (r *IterationPostgres) ListIterationsBySession(ctx context.Context, sessionID string) ([]*entity.Iteration, error) {
	sessID, err := uuid.Parse(sessionID)
	if err != nil {
		return nil, fmt.Errorf("invalid session ID: %w", err)
	}

	dbIters, err := r.queries.ListIterationsBySession(ctx, pgtype.UUID{
		Bytes: sessID,
		Valid: true,
	})
	if err != nil {
		return nil, fmt.Errorf("list iterations: %w", err)
	}

	iterations := make([]*entity.Iteration, len(dbIters))
	for i, dbIter := range dbIters {
		iterations[i] = toEntityIteration(&dbIter)
	}

	return iterations, nil
}

func (r *IterationPostgres) GetNextIteration(ctx context.Context, sessionID string) (*entity.Iteration, error) {
	sessID, err := uuid.Parse(sessionID)
	if err != nil {
		return nil, fmt.Errorf("invalid session ID: %w", err)
	}

	dbIter, err := r.queries.GetNextIteration(ctx, pgtype.UUID{
		Bytes: sessID,
		Valid: true,
	})
	if err != nil {
		return nil, fmt.Errorf("get next unanswered iteration: %w", err)
	}

	return toEntityIteration(&dbIter), nil
}

func (r *IterationPostgres) GetCurrentIteration(ctx context.Context, sessionID string) (*entity.Iteration, error) {
	sessID, err := uuid.Parse(sessionID)
	if err != nil {
		return nil, fmt.Errorf("invalid session ID: %w", err)
	}

	dbIter, err := r.queries.GetCurrentIteration(ctx, pgtype.UUID{
		Bytes: sessID,
		Valid: true,
	})
	if err != nil {
		return nil, fmt.Errorf("get next unanswered iteration: %w", err)
	}

	return toEntityIteration(&dbIter), nil
}

// GetMaxIterationNumber returns the maximum iteration_number for a session
func (r *IterationPostgres) GetMaxIterationNumber(ctx context.Context, sessionID string) (int, error) {
	sessID, err := uuid.Parse(sessionID)
	if err != nil {
		return 0, fmt.Errorf("invalid session ID: %w", err)
	}

	// Query: SELECT MAX(iteration_number) FROM session_iterations WHERE session_id = $1
	query := `SELECT COALESCE(MAX(iteration_number), 0) FROM session_iterations WHERE session_id = $1`

	var maxNumber int32
	err = r.db.QueryRow(ctx, query, pgtype.UUID{
		Bytes: sessID,
		Valid: true,
	}).Scan(&maxNumber)

	if err != nil {
		return 0, fmt.Errorf("query max iteration number: %w", err)
	}

	return int(maxNumber), nil
}
