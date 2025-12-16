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

// SessionMessageRepository defines the interface for session draft messages persistence
type SessionMessageRepository interface {
	CreateMessage(ctx context.Context, sessionID, messageText string) (*entity.SessionMessage, error)
	GetSessionMessages(ctx context.Context, sessionID string) ([]*entity.SessionMessage, error)
	DeleteSessionMessages(ctx context.Context, sessionID string) error
}

var _ SessionMessageRepository = &SessionMessagePostgres{}

// SessionMessagePostgres implements SessionMessageRepository using PostgreSQL
type SessionMessagePostgres struct {
	db      *pgxpool.Pool
	queries *sqlc.Queries
}

func NewSessionMessagePostgres(db *pgxpool.Pool) *SessionMessagePostgres {
	return &SessionMessagePostgres{
		db:      db,
		queries: sqlc.New(db),
	}
}

func (r *SessionMessagePostgres) CreateMessage(
	ctx context.Context,
	sessionID string,
	messageText string,
) (*entity.SessionMessage, error) {
	sessID, err := uuid.Parse(sessionID)
	if err != nil {
		return nil, fmt.Errorf("invalid session ID: %w", err)
	}

	dbMsg, err := r.queries.CreateSessionMessage(ctx, sqlc.CreateSessionMessageParams{
		SessionID: pgtype.UUID{
			Bytes: sessID,
			Valid: true,
		},
		MessageText: messageText,
	})
	if err != nil {
		return nil, fmt.Errorf("create session message: %w", err)
	}

	return toEntitySessionMessage(&dbMsg), nil
}

func (r *SessionMessagePostgres) GetSessionMessages(
	ctx context.Context,
	sessionID string,
) ([]*entity.SessionMessage, error) {
	sessID, err := uuid.Parse(sessionID)
	if err != nil {
		return nil, fmt.Errorf("invalid session ID: %w", err)
	}

	dbMsgs, err := r.queries.GetSessionMessages(ctx, pgtype.UUID{
		Bytes: sessID,
		Valid: true,
	})
	if err != nil {
		return nil, fmt.Errorf("get session messages: %w", err)
	}

	messages := make([]*entity.SessionMessage, 0, len(dbMsgs))
	for i := range dbMsgs {
		messages = append(messages, toEntitySessionMessage(&dbMsgs[i]))
	}

	return messages, nil
}

func (r *SessionMessagePostgres) DeleteSessionMessages(ctx context.Context, sessionID string) error {
	sessID, err := uuid.Parse(sessionID)
	if err != nil {
		return fmt.Errorf("invalid session ID: %w", err)
	}

	if err := r.queries.DeleteSessionMessages(ctx, pgtype.UUID{
		Bytes: sessID,
		Valid: true,
	}); err != nil {
		return fmt.Errorf("delete session messages: %w", err)
	}

	return nil
}
