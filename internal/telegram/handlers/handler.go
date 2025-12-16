package handlers

import (
	"context"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Handler state constants
const (
	HandlerStateCallback              = "CALLBACK"
	HandlerStateAskGoal               = "ASK_USER_GOAL"
	HandlerStateAskContext            = "ASK_USER_CONTEXT"
	HandlerStateWaitingAnswers        = "WAITING_FOR_ANSWERS"
	HandlerStateDraftCollecting       = "DRAFT_COLLECTING"
	HandlerStateAskProjectName        = "ASK_PROJECT_NAME"
	HandlerStateAskProjectDescription = "ASK_PROJECT_DESCRIPTION"
)

// Message represents a normalized Telegram message
type Message struct {
	ChatID       int64
	UserID       int64
	MessageID    int
	Text         string
	Voice        *tgbotapi.Voice
	Document     *tgbotapi.Document
	CallbackData string
	CallbackID   string
}

// Handler defines the interface for state-specific handlers
type Handler interface {
	// Handle processes a message for this state
	Handle(ctx context.Context, msg *Message) error

	// GetState returns the state this handler manages
	GetState() string
}

// BaseHandler provides common functionality for all handlers
type BaseHandler struct {
	stateName     string
	messageSender *MessageSender
}

// GetState implements Handler
func (h *BaseHandler) GetState() string {
	return h.stateName
}

// sendMessage is a convenience wrapper for messageSender.Send
func (h *BaseHandler) sendMessage(chatID int64, text string, markup interface{}) {
	if h.messageSender != nil {
		h.messageSender.Send(chatID, text, markup)
	}
}

// validStates defines all valid handler states
var validStates = map[string]bool{
	HandlerStateCallback:              true,
	HandlerStateAskGoal:               true,
	HandlerStateAskContext:            true,
	HandlerStateWaitingAnswers:        true,
	HandlerStateDraftCollecting:       true,
	HandlerStateAskProjectName:        true,
	HandlerStateAskProjectDescription: true,
}

// IsValidState checks if a state is valid for handler registration
func IsValidState(state string) bool {
	_, ok := validStates[state]
	return ok
}
