package session

import (
	"github.com/go-chi/chi/v5"
)

// RegisterRoutes registers session routes
func RegisterRoutes(r chi.Router, h *Handler) {
	r.Route("/interview-session", func(r chi.Router) {
		r.Post("/", h.StartSession)
		r.Get("/{id}", h.GetSession)
		r.Post("/{id}/answer/{question_id}", h.SubmitTextAnswer)
		r.Post("/{id}/answer/audio/{question_id}", h.SubmitAudioAnswer)
		r.Get("/{id}/result", h.GetSessionResult)
		r.Post("/{id}/cancel", h.CancelSession)
	})
}
