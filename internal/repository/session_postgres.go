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

// SessionRepository defines the interface for session persistence
type SessionRepository interface {
	CreateSession(ctx context.Context, session entity.Session) (*entity.Session, error)
	CreateFilledSession(ctx context.Context, session *entity.Session) (*entity.Session, error)
	GetSessionByID(ctx context.Context, id string) (*entity.Session, error)
	AquireSessionByID(ctx context.Context, id string) (*entity.Session, error)
	UpdateSessionStatus(ctx context.Context, id string, status entity.SessionStatus) (*entity.Session, error)
	UpdateSessionIteration(ctx context.Context, id string) (*entity.Session, error)
	ResetSessionIteration(ctx context.Context, id string) (*entity.Session, error)
	UpdateSessionProjectContext(ctx context.Context, id, projectCtx string) (*entity.Session, error)
	UpdateSessionRAGProjectContext(ctx context.Context, sessionID, projectID, projectCtx string) (*entity.Session, error)
	UpdateSessionUserGoal(ctx context.Context, id, userGoal string) (*entity.Session, error)
	UpdateSessionType(ctx context.Context, id string, sessionType entity.SessionType) (*entity.Session, error)
	UpdateSessionResult(ctx context.Context, id string, status entity.SessionStatus, result, err *string) (
		*entity.Session, error,
	)
	DeleteSession(ctx context.Context, id string) error
}

var _ SessionRepository = &SessionPostgres{}

// SessionPostgres implements SessionRepository using PostgreSQL
type SessionPostgres struct {
	db      *pgxpool.Pool
	queries *sqlc.Queries
}

func NewSessionPostgres(db *pgxpool.Pool) *SessionPostgres {
	return &SessionPostgres{
		db:      db,
		queries: sqlc.New(db),
	}
}

func (r *SessionPostgres) CreateSession(ctx context.Context, session entity.Session) (*entity.Session, error) {
	sessionID, err := uuid.Parse(session.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid session ID: %w", err)
	}

	params := sqlc.CreateSessionParams{
		ID: pgtype.UUID{
			Bytes: sessionID,
			Valid: true,
		},
		Status: string(session.Status),
	}

	dbSession, err := r.queries.CreateSession(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	return toEntitySession(&dbSession), nil
}

func (r *SessionPostgres) CreateFilledSession(ctx context.Context, session *entity.Session) (*entity.Session, error) {
	sessionID, err := uuid.Parse(session.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid session ID: %w", err)
	}

	params := sqlc.CreateFilledSessionParams{
		ID: pgtype.UUID{
			Bytes: sessionID,
			Valid: true,
		},
		Status: string(session.Status),
	}

	// Set optional project_id
	if session.ProjectID != nil && *session.ProjectID != "" {
		projectUUID, err := uuid.Parse(*session.ProjectID)
		if err != nil {
			return nil, fmt.Errorf("invalid project ID: %w", err)
		}
		params.ProjectID = pgtype.UUID{
			Bytes: projectUUID,
			Valid: true,
		}
	}

	// Set optional type
	if session.Type != nil {
		params.Type = pgtype.Text{
			String: string(*session.Type),
			Valid:  true,
		}
	}

	// Set optional user_goal
	if session.UserGoal != nil {
		params.UserGoal = pgtype.Text{
			String: *session.UserGoal,
			Valid:  true,
		}
	}

	// Set optional project_context
	if session.ProjectContext != nil {
		params.ProjectContext = pgtype.Text{
			String: *session.ProjectContext,
			Valid:  true,
		}
	}

	dbSession, err := r.queries.CreateFilledSession(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("create filled session: %w", err)
	}

	return toEntitySession(&dbSession), nil
}

func (r *SessionPostgres) GetSessionByID(ctx context.Context, id string) (*entity.Session, error) {
	sessionID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid session ID: %w", err)
	}

	dbSession, err := r.queries.GetSessionByID(ctx, pgtype.UUID{
		Bytes: sessionID,
		Valid: true,
	})
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}

	return toEntitySession(&dbSession), nil
}

func (r *SessionPostgres) AquireSessionByID(ctx context.Context, id string) (*entity.Session, error) {
	sessionID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid session ID: %w", err)
	}

	dbSession, err := r.queries.AquireSessionByID(ctx, pgtype.UUID{
		Bytes: sessionID,
		Valid: true,
	})
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}

	return toEntitySession(&dbSession), nil
}

func (r *SessionPostgres) UpdateSessionStatus(ctx context.Context, id string, status entity.SessionStatus) (
	*entity.Session, error,
) {
	sessionID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid session ID: %w", err)
	}

	dbSession, err := r.queries.UpdateSessionStatus(ctx, sqlc.UpdateSessionStatusParams{
		ID: pgtype.UUID{
			Bytes: sessionID,
			Valid: true,
		},
		Status: string(status),
	})
	if err != nil {
		return nil, fmt.Errorf("update session status: %w", err)
	}

	return toEntitySession(&dbSession), nil
}

func (r *SessionPostgres) UpdateSessionIteration(ctx context.Context, id string) (*entity.Session, error) {
	sessionID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid session ID: %w", err)
	}

	dbSession, err := r.queries.UpdateSessionIteration(ctx, pgtype.UUID{
		Bytes: sessionID,
		Valid: true,
	})
	if err != nil {
		return nil, fmt.Errorf("update session status: %w", err)
	}

	return toEntitySession(&dbSession), nil
}

func (r *SessionPostgres) ResetSessionIteration(ctx context.Context, id string) (*entity.Session, error) {
	sessionID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid session ID: %w", err)
	}

	dbSession, err := r.queries.ResetSessionIteration(ctx, pgtype.UUID{
		Bytes: sessionID,
		Valid: true,
	})
	if err != nil {
		return nil, fmt.Errorf("update session status: %w", err)
	}

	return toEntitySession(&dbSession), nil
}

func (r *SessionPostgres) UpdateSessionProjectContext(ctx context.Context, sessionID, projectCtx string) (
	*entity.Session, error,
) {
	sID, err := uuid.Parse(sessionID)
	if err != nil {
		return nil, fmt.Errorf("invalid session ID: %w", err)
	}

	dbSession, err := r.queries.UpdateSessionProjectContext(ctx, sqlc.UpdateSessionProjectContextParams{
		ID: pgtype.UUID{
			Bytes: sID,
			Valid: true,
		},
		ProjectContext: pgtype.Text{
			String: projectCtx,
			Valid:  true,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("update project contex: %w", err)
	}

	return toEntitySession(&dbSession), nil
}

func (r *SessionPostgres) UpdateSessionResult(
	ctx context.Context, id string, status entity.SessionStatus, result, errRes *string,
) (*entity.Session, error) {
	sessionID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid session ID: %w", err)
	}

	params := sqlc.UpdateSessionResultParams{
		ID: pgtype.UUID{
			Bytes: sessionID,
			Valid: true,
		},
		Status: string(status),
	}

	if result != nil {
		params.Result = pgtype.Text{
			Valid:  true,
			String: *result,
		}
	}

	if errRes != nil {
		params.Error = pgtype.Text{
			Valid:  true,
			String: *errRes,
		}
	}

	session, err := r.queries.UpdateSessionResult(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("update session status: %w", err)
	}

	return toEntitySession(&session), nil
}

func (r *SessionPostgres) UpdateSessionRAGProjectContext(ctx context.Context, sessionID, projectID, projectCtx string) (*entity.Session, error) {
	sessionUUID, err := uuid.Parse(sessionID)
	if err != nil {
		return nil, fmt.Errorf("invalid session ID: %w", err)
	}

	projectUUID, err := uuid.Parse(projectID)
	if err != nil {
		return nil, fmt.Errorf("invalid project ID: %w", err)
	}

	dbSession, err := r.queries.UpdateSessionRAGProjectContext(ctx, sqlc.UpdateSessionRAGProjectContextParams{
		ProjectContext: pgtype.Text{
			String: projectCtx,
			Valid:  projectCtx != "",
		},
		ID: pgtype.UUID{
			Bytes: sessionUUID,
			Valid: true,
		},
		ProjectID: pgtype.UUID{
			Bytes: projectUUID,
			Valid: true,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("update rag project context: %w", err)
	}

	return toEntitySession(&dbSession), nil
}

func (r *SessionPostgres) UpdateSessionUserGoal(ctx context.Context, id, userGoal string) (*entity.Session, error) {
	sessionID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid session ID: %w", err)
	}

	dbSession, err := r.queries.UpdateSessionUserGoal(ctx, sqlc.UpdateSessionUserGoalParams{
		ID: pgtype.UUID{
			Bytes: sessionID,
			Valid: true,
		},
		UserGoal: pgtype.Text{
			String: userGoal,
			Valid:  true,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("update user goal: %w", err)
	}

	return toEntitySession(&dbSession), nil
}

func (r *SessionPostgres) UpdateSessionType(ctx context.Context, id string, sessionType entity.SessionType) (*entity.Session, error) {
	sessionID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid session ID: %w", err)
	}

	dbSession, err := r.queries.UpdateSessionType(ctx, sqlc.UpdateSessionTypeParams{
		ID: pgtype.UUID{
			Bytes: sessionID,
			Valid: true,
		},
		Type: pgtype.Text{
			String: string(sessionType),
			Valid:  true,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("update session type: %w", err)
	}

	return toEntitySession(&dbSession), nil
}

func (r *SessionPostgres) DeleteSession(ctx context.Context, id string) error {
	sessionID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid session ID: %w", err)
	}

	err = r.queries.DeleteSession(ctx, pgtype.UUID{
		Bytes: sessionID,
		Valid: true,
	})
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}

	return nil
}
