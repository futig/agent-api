package entity

import (
	"fmt"
	"time"
)

type SessionStatus string

// Session status represents the current state of the interview/draft session workflow
const (
	// Initial states
	SessionStatusNew SessionStatus = "NEW" // Session created, waiting for user goal

	// Project context setup
	SessionStatusAskUserGoal           SessionStatus = "ASK_USER_GOAL"            // Requesting project description
	SessionStatusSelectOrCreateProject SessionStatus = "SELECT_OR_CREATE_PROJECT" // Choose existing project or create new
	SessionStatusAskUserContext        SessionStatus = "ASK_USER_CONTEXT"         // Manual context questions (if no project)

	// Mode selection
	SessionStatusChooseMode    SessionStatus = "CHOOSE_MODE"    // Select Interview or Draft mode
	SessionStatusInterviewInfo SessionStatus = "INTERVIEW_INFO" // Explain interview format
	SessionStatusDraftInfo     SessionStatus = "DRAFT_INFO"     // Explain draft format

	// Question generation and interview
	SessionStatusGeneratingQuestions SessionStatus = "GENERATING_QUESTIONS" // Generating questions via LLM
	SessionStatusWaitingForAnswers   SessionStatus = "WAITING_FOR_ANSWERS"  // Interview questions - waiting for user answers
	SessionStatusDraftCollecting     SessionStatus = "DRAFT_COLLECTING"     // Collecting draft materials (up to 10 messages)

	// Processing and validation
	SessionStatusValidating             SessionStatus = "VALIDATING"              // Validating answers
	SessionStatusGeneratingRequirements SessionStatus = "GENERATING_REQUIREMENTS" // Generating business requirements

	// Final states
	SessionStatusDone     SessionStatus = "DONE"     // Session completed successfully
	SessionStatusError    SessionStatus = "ERROR"    // Session failed with error
	SessionStatusCanceled SessionStatus = "CANCELED" // Session cancelled by user

	// Project save states
	SessionStatusAskProjectName        SessionStatus = "ASK_PROJECT_NAME"        // Asking for new project name
	SessionStatusAskProjectDescription SessionStatus = "ASK_PROJECT_DESCRIPTION" // Asking for new project description
)

type SessionType string

const (
	SessionTypeDraft     SessionType = "DRAFT"
	SessionTypeInterview SessionType = "INTERVIEW"
)

func (st *SessionType) Validate() error {
	switch *st {
	case SessionTypeDraft, SessionTypeInterview:
		return nil
	default:
		return fmt.Errorf("unknown session type: %s", *st)
	}
}

type QuestionStatus string

const (
	AnswerStatusUnanswered QuestionStatus = "UNANSWERED"
	AnswerStatusSkiped     QuestionStatus = "SKIPED"
	AnswerStatusAnswered   QuestionStatus = "ANSWERED"
)

type Session struct {
	ID               string        `json:"session_id"`
	ProjectID        *string       `json:"project_id,omitempty"`
	Status           SessionStatus `json:"session_status"`
	Type             *SessionType  `json:"session_type,omitempty"`
	UserGoal         *string       `json:"user_goal,omitempty"`
	ProjectContext   *string       `json:"project_context,omitempty"`
	CurrentIteration int           `json:"iteration_number"`
	Result           *string       `json:"final_result,omitempty"`
	Error            *string       `json:"error,omitempty"`
	CreatedAt        time.Time     `json:"created_at"`
	UpdatedAt        time.Time     `json:"updated_at"`
}

type Iteration struct {
	ID              string    `json:"id"`
	SessionID       string    `json:"session_id"`
	IterationNumber int       `json:"iteration_number"`
	Title           string    `json:"title"`
	CreatedAt       time.Time `json:"created_at"`
}

type Question struct {
	ID             string         `json:"id"`
	IterationID    string         `json:"iteration_id"`
	QuestionNumber int            `json:"question_number"`
	Status         QuestionStatus `json:"status"`
	Question       string         `json:"question"`
	Explanation    string         `json:"explanation"`
	Answer         *string        `json:"answer,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	AnsweredAt     *time.Time     `json:"answered_at,omitempty"`
}

type Project struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	Files       []*File   `json:"files,omitempty"`
}

type File struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"project_id"`
	Filename    string    `json:"name"`
	Size        int64     `json:"size"`
	ContentType string    `json:"content_type"`
	CreatedAt   time.Time `json:"created_at"`
}

// SessionMessage represents a draft message in a session
type SessionMessage struct {
	ID          string    `json:"id"`
	SessionID   string    `json:"session_id"`
	MessageText string    `json:"message_text"`
	CreatedAt   time.Time `json:"created_at"`
}
