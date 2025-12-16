package entity

import (
	"mime/multipart"
	"time"
)

type StartSessionRequest struct {
	ProjectID        *string              `json:"project_id,omitempty"`
	UserGoal         string               `json:"user_goal"`
	ContextQuestions []QuestionWithAnswer `json:"context_questions,omitempty"`
	CallbackURL      string               `json:"callback_url,omitempty"`
}

type SubmitAnswerRequest struct {
	Answer      string `json:"answers"`
	IsSkipped   bool   `json:"is_skipped"`
	CallbackURL string `json:"callback_url"`
}

type SubmitAudioAnswerRequest struct {
	AudioFile   *multipart.FileHeader
	IsSkipped   bool   `json:"is_skipped"`
	CallbackURL string `json:"callback_url"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

type QuestionDTO struct {
	ID             string         `json:"id"`
	QuestionNumber int            `json:"question_number"`
	Status         QuestionStatus `json:"status"`
	Question       string         `json:"question"`
	Explanation    string         `json:"explanation"`
}

type IterationWithQuestions struct {
	SessionID       string        `json:"session_id"`
	IterationID     string        `json:"iteration_id"`
	IterationNumber int           `json:"iteration_number"`
	Title           string        `json:"title"`
	Questions       []QuestionDTO `json:"questions"`
}

type SessionDTO struct {
	ID               string        `json:"session_id"`
	ProjectID        *string       `json:"project_id,omitempty"`
	Status           SessionStatus `json:"session_status"`
	CurrentIteration int           `json:"iteration_number"`
	Result           *string       `json:"final_result,omitempty"`
	Error            *string       `json:"error,omitempty"`
	CreatedAt        time.Time     `json:"created_at"`
	UpdatedAt        time.Time     `json:"updated_at"`
}
