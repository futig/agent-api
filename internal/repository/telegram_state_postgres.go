package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/futig/agent-backend/internal/repository/sqlc"
	"github.com/futig/agent-backend/internal/telegram/state"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TelegramSessionRepository handles telegram session mapping persistence
type TelegramSessionRepository struct {
	db      *pgxpool.Pool
	queries *sqlc.Queries
}

// NewTelegramStateRepository creates a new telegram session repository
func NewTelegramStateRepository(db *pgxpool.Pool) *TelegramSessionRepository {
	return &TelegramSessionRepository{
		db:      db,
		queries: sqlc.New(db),
	}
}

// Get retrieves telegram session by user ID
func (r *TelegramSessionRepository) Get(ctx context.Context, userID int64) (*state.TelegramSession, error) {
	dbSession, err := r.queries.GetTelegramSession(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("telegram session not found: %d", userID)
		}
		return nil, fmt.Errorf("query telegram session: %w", err)
	}

	return toStateTelegramSession(&dbSession), nil
}

// GetWithSession retrieves telegram session with joined session data by user ID
func (r *TelegramSessionRepository) GetWithSession(ctx context.Context, userID int64) (*state.TelegramSessionWithSession, error) {
	row, err := r.queries.GetTelegramSessionWithSession(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("telegram session not found: %d", userID)
		}
		return nil, fmt.Errorf("query telegram session with session: %w", err)
	}

	return toStateTelegramSessionWithSession(&row), nil
}

// Set saves telegram session
func (r *TelegramSessionRepository) Set(ctx context.Context, telegramSession *state.TelegramSession) error {
	params := toDBUpsertParams(telegramSession)

	err := r.queries.UpsertTelegramSession(ctx, params)
	if err != nil {
		return fmt.Errorf("upsert telegram session: %w", err)
	}

	return nil
}

// Delete removes telegram session
func (r *TelegramSessionRepository) Delete(ctx context.Context, userID int64) error {
	err := r.queries.DeleteTelegramSession(ctx, userID)
	if err != nil {
		return fmt.Errorf("delete telegram session: %w", err)
	}

	return nil
}

// GetBySessionID retrieves telegram session by session ID
func (r *TelegramSessionRepository) GetBySessionID(ctx context.Context, sessionID string) (*state.TelegramSession, error) {
	parsedUUID, err := uuid.Parse(sessionID)
	if err != nil {
		return nil, fmt.Errorf("invalid session ID format: %w", err)
	}

	var sessionUUID pgtype.UUID
	sessionUUID.Bytes = parsedUUID
	sessionUUID.Valid = true

	dbSession, err := r.queries.GetTelegramSessionBySessionID(ctx, sessionUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("telegram session not found for session: %s", sessionID)
		}
		return nil, fmt.Errorf("query telegram session by session: %w", err)
	}

	return toStateTelegramSession(&dbSession), nil
}

// toStateTelegramSession converts from sqlc TelegramSession to state.TelegramSession
func toStateTelegramSession(dbSession *sqlc.TelegramSession) *state.TelegramSession {
	telegramSession := &state.TelegramSession{
		UserID:    dbSession.UserID,
		CreatedAt: dbSession.CreatedAt.Time,
		UpdatedAt: dbSession.UpdatedAt.Time,
	}

	// Convert UUID to string
	if dbSession.SessionID.Valid {
		telegramSession.SessionID = uuid.UUID(dbSession.SessionID.Bytes).String()
	}

	// Set state data
	if len(dbSession.StateData) > 0 {
		telegramSession.StateData = json.RawMessage(dbSession.StateData)
	} else {
		telegramSession.StateData = json.RawMessage("{}")
	}

	return telegramSession
}

// toDBUpsertParams converts from state.TelegramSession to sqlc UpsertTelegramSessionParams
func toDBUpsertParams(telegramSession *state.TelegramSession) sqlc.UpsertTelegramSessionParams {
	params := sqlc.UpsertTelegramSessionParams{
		UserID:    telegramSession.UserID,
		CreatedAt: pgtype.Timestamp{Time: telegramSession.CreatedAt, Valid: true},
		UpdatedAt: pgtype.Timestamp{Time: telegramSession.UpdatedAt, Valid: true},
	}

	// Convert UUID string to pgtype.UUID
	if telegramSession.SessionID != "" {
		parsedUUID, err := uuid.Parse(telegramSession.SessionID)
		if err == nil {
			params.SessionID.Bytes = parsedUUID
			params.SessionID.Valid = true
		}
	}

	// Convert state data
	stateData := []byte(telegramSession.StateData)
	if len(stateData) == 0 {
		stateData = []byte("{}")
	}
	params.StateData = stateData

	return params
}

// toStateTelegramSessionWithSession converts from sqlc GetTelegramSessionWithSessionRow to state.TelegramSessionWithSession
func toStateTelegramSessionWithSession(row *sqlc.GetTelegramSessionWithSessionRow) *state.TelegramSessionWithSession {
	result := &state.TelegramSessionWithSession{
		TelegramSession: &state.TelegramSession{
			UserID:    row.UserID,
			CreatedAt: row.TgCreatedAt.Time,
			UpdatedAt: row.TgUpdatedAt.Time,
		},
	}

	// Convert telegram session_id
	if row.SessionID.Valid {
		result.TelegramSession.SessionID = uuid.UUID(row.SessionID.Bytes).String()
	}

	// Set state data
	if len(row.StateData) > 0 {
		result.TelegramSession.StateData = json.RawMessage(row.StateData)
	} else {
		result.TelegramSession.StateData = json.RawMessage("{}")
	}

	// Convert session fields (they're nullable because of LEFT JOIN)
	if row.SessionIDFull.Valid {
		result.SessionID = uuid.UUID(row.SessionIDFull.Bytes).String()
	}

	if row.SessionStatus.Valid {
		result.SessionStatus = row.SessionStatus.String
	}

	if row.SessionType.Valid {
		result.SessionType = row.SessionType.String
	}

	if row.SessionProjectID.Valid {
		result.ProjectID = uuid.UUID(row.SessionProjectID.Bytes).String()
	}

	return result
}
