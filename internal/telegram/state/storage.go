package state

import (
	"context"
	"encoding/json"
	"time"
)

// TelegramSession represents telegram user -> session mapping with UI state
type TelegramSession struct {
	UserID    int64           `json:"user_id"`
	SessionID string          `json:"session_id,omitempty"`
	StateData json.RawMessage `json:"state_data,omitempty"` // Telegram-specific UI state
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// TelegramSessionWithSession contains telegram session with joined session data
type TelegramSessionWithSession struct {
	TelegramSession *TelegramSession
	SessionID       string // Empty if no active session
	SessionStatus   string // Empty if no active session
	SessionType     string // Empty if no active session
	ProjectID       string // Empty if no active session or no project
}

// StateData contains telegram-specific UI state (stored in StateData JSONB)
// Version 1: Initial implementation
type StateData struct {
	// Version for compatibility tracking (current version: 1)
	Version int `json:"version,omitempty"`

	// Context question tracking
	CurrentQuestionIndex int `json:"current_question_index,omitempty"`

	// Interview/Draft tracking
	CurrentIterationID string `json:"current_iteration_id,omitempty"`
	CurrentQuestionID  string `json:"current_question_id,omitempty"`
	DraftMessageCount  int    `json:"draft_message_count,omitempty"`
	// Skipped questions flow tracking
	AnsweringSkipped             bool     `json:"answering_skipped,omitempty"`
	TotalSkippedQuestions        int      `json:"total_skipped_questions,omitempty"`        // Total count when starting skipped flow
	CurrentSkippedQuestionNumber int      `json:"current_skipped_question_number,omitempty"` // Current position in skipped flow (1-based)
	SkippedQuestionIDs           []string `json:"skipped_question_ids,omitempty"`            // List of all skipped question IDs
	CurrentSkippedQuestionIndex  int      `json:"current_skipped_question_index,omitempty"`  // Current index in SkippedQuestionIDs (0-based)
	// Question history tracking (for back/forward navigation)
	// Only one step back allowed
	PreviousQuestionID string   `json:"previous_question_id,omitempty"` // Previous question ID (only one level back)
	NextQuestionIDs    []string `json:"next_question_ids,omitempty"`    // Stack for going forward after answering

	// Project selection tracking
	ProjectID         string `json:"project_id,omitempty"`
	ProjectListPage   int    `json:"project_list_page,omitempty"`
	ProjectListOffset int    `json:"project_list_offset,omitempty"`

	// Project creation tracking (for save-to-new-project flow)
	ProjectName string `json:"project_name,omitempty"`

	// Last message ID (for editing)
	LastMessageID int `json:"last_message_id,omitempty"`

	// Processing state (for idempotency)
	IsProcessing      bool      `json:"is_processing,omitempty"`
	ProcessingStarted time.Time `json:"processing_started,omitempty"`

	// Confirmation for destructive actions
	PendingConfirmation string `json:"pending_confirmation,omitempty"` // "cancel", "finish"
}

const (
	// StateDataCurrentVersion is the current version of StateData
	StateDataCurrentVersion = 1
)

// Storage defines the interface for telegram session persistence
type Storage interface {
	// Get retrieves telegram session by user ID
	Get(ctx context.Context, userID int64) (*TelegramSession, error)

	// GetWithSession retrieves telegram session with joined session data by user ID
	GetWithSession(ctx context.Context, userID int64) (*TelegramSessionWithSession, error)

	// Set saves telegram session
	Set(ctx context.Context, session *TelegramSession) error

	// Delete removes telegram session
	Delete(ctx context.Context, userID int64) error

	// GetBySessionID retrieves telegram session by session ID
	GetBySessionID(ctx context.Context, sessionID string) (*TelegramSession, error)
}
