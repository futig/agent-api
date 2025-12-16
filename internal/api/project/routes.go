package project

import (
	"github.com/go-chi/chi/v5"
)

// RegisterRoutes registers project routes
func RegisterRoutes(r chi.Router, h *Handler) {
	r.Route("/projects", func(r chi.Router) {
		r.Post("/", h.CreateProject)
		r.Get("/", h.ListProjects)

		r.Route("/{project_id}", func(r chi.Router) {
			r.Get("/", h.GetProject)
			r.Delete("/", h.DeleteProject)
			r.Post("/", h.AddFiles)
			r.Get("/files", h.ListFiles)
		})
	})
}
