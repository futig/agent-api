package entity

// CallbackEventType represents the type of callback event
type CallbackEventType string

const (
	CallbackEventTypeQuestions      CallbackEventType = "questions"
	CallbackEventTypeProjectUpdated CallbackEventType = "projectUpdated"
	CallbackEventTypeFinalResult    CallbackEventType = "finalResult"
	CallbackEventTypeError          CallbackEventType = "error"
)

// CallbackEvent represents a callback event
type CallbackEvent struct {
	Event     CallbackEventType `json:"event"`
	Timestamp string            `json:"timestamp"` // ISO-8601 UTC
	Data      any               `json:"data"`
}

// CallbackProjectUpdatedData represents data for project updated event
type CallbackProjectUpdatedData struct {
	ID          string             `json:"id"`
	Title       string             `json:"title"`
	Description string             `json:"description"`
	Size        int64              `json:"size"`
	Files       []CallbackFileInfo `json:"files"`
}

// CallbackFileInfo represents file information in project updated event
type CallbackFileInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Size int64  `json:"size"`
}

// CallbackErrorData represents data for error event
type CallbackErrorData struct {
	Error CallbackErrorDetails `json:"error"`
}

// CallbackErrorDetails contains error information
type CallbackErrorDetails struct {
	Message string         `json:"message"`
	Details map[string]any `json:"details"` // Context like ids, files
}
