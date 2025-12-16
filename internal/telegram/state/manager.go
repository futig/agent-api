package state

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// contextKey is a type for context keys to avoid collisions
type contextKey string

const stateDataKey contextKey = "state_data"

// StateDataFromContext retrieves StateData from context if available
func StateDataFromContext(ctx context.Context) (*StateData, bool) {
	data, ok := ctx.Value(stateDataKey).(*StateData)
	return data, ok
}

// ContextWithStateData attaches StateData to context for request-scoped caching
func ContextWithStateData(ctx context.Context, data *StateData) context.Context {
	return context.WithValue(ctx, stateDataKey, data)
}

// Manager manages telegram sessions
type Manager struct {
	storage Storage
}

// NewManager creates a new state manager
func NewManager(storage Storage) *Manager {
	return &Manager{
		storage: storage,
	}
}

// GetSession retrieves telegram session from storage
func (m *Manager) GetSession(ctx context.Context, userID int64) (*TelegramSession, error) {
	session, err := m.storage.Get(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get telegram session from storage: %w", err)
	}

	return session, nil
}

// GetSessionWithSession retrieves telegram session with joined session data from storage
func (m *Manager) GetSessionWithSession(ctx context.Context, userID int64) (*TelegramSessionWithSession, error) {
	result, err := m.storage.GetWithSession(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get telegram session with session from storage: %w", err)
	}

	return result, nil
}

// SetSession saves telegram session to storage
func (m *Manager) SetSession(ctx context.Context, session *TelegramSession) error {
	session.UpdatedAt = time.Now()

	if err := m.storage.Set(ctx, session); err != nil {
		return fmt.Errorf("save telegram session to storage: %w", err)
	}

	return nil
}

// DeleteSession removes telegram session from storage
func (m *Manager) DeleteSession(ctx context.Context, userID int64) error {
	if err := m.storage.Delete(ctx, userID); err != nil {
		return fmt.Errorf("delete telegram session from storage: %w", err)
	}

	return nil
}

// GetStateData extracts typed state data
// First checks context for cached data, then loads from storage if needed
func (m *Manager) GetStateData(ctx context.Context, userID int64) (*StateData, error) {
	// Check context cache first
	if data, ok := StateDataFromContext(ctx); ok {
		return data, nil
	}

	// Load from storage
	session, err := m.GetSession(ctx, userID)
	if err != nil {
		return nil, err
	}

	if len(session.StateData) == 0 {
		// Return new StateData with current version
		return &StateData{
			Version: StateDataCurrentVersion,
		}, nil
	}

	var data StateData
	if err := json.Unmarshal(session.StateData, &data); err != nil {
		return nil, fmt.Errorf("unmarshal state data: %w", err)
	}

	// Auto-upgrade from old versions without version field
	if data.Version == 0 {
		data.Version = StateDataCurrentVersion
	}

	return &data, nil
}

// UpdateStateData updates state data
func (m *Manager) UpdateStateData(ctx context.Context, userID int64, data *StateData) error {
	session, err := m.GetSession(ctx, userID)
	if err != nil {
		return err
	}

	// Ensure version is set to current version
	data.Version = StateDataCurrentVersion

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal state data: %w", err)
	}

	session.StateData = jsonData
	return m.SetSession(ctx, session)
}

// CreateOrUpdateSession creates new telegram session or updates existing
func (m *Manager) CreateOrUpdateSession(ctx context.Context, userID int64, sessionID string) error {
	session, err := m.GetSession(ctx, userID)
	if err != nil {
		// Create new session mapping
		session = &TelegramSession{
			UserID:    userID,
			SessionID: sessionID,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			StateData: json.RawMessage("{}"),
		}
	} else {
		// Update existing
		if sessionID != "" {
			session.SessionID = sessionID
		}
		session.UpdatedAt = time.Now()
	}

	return m.SetSession(ctx, session)
}

// GetBySessionID retrieves telegram session by session ID
func (m *Manager) GetBySessionID(ctx context.Context, sessionID string) (*TelegramSession, error) {
	return m.storage.GetBySessionID(ctx, sessionID)
}
