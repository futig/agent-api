package docs

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	httpSwagger "github.com/swaggo/http-swagger/v2"
)

// Handler returns a handler that serves Swagger UI.
func Handler() http.HandlerFunc {
	return httpSwagger.Handler(
		httpSwagger.URL("/docs/swagger.yaml"),
		httpSwagger.DeepLinking(true),
		httpSwagger.DocExpansion("list"),
		httpSwagger.DomID("swagger-ui"),
	)
}

// SwaggerYAMLHandler serves the Swagger YAML specification file.
func SwaggerYAMLHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "docs/swagger.yaml")
	}
}

// RegisterRoutes registers Swagger documentation routes on the router.
func RegisterRoutes(r chi.Router) {
	// Redirect base /docs to the Swagger UI index
	r.Get("/docs", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/docs/index.html", http.StatusFound)
	})

	// Serve Swagger UI and assets under /docs/*
	r.Get("/docs/*", Handler())

	// Serve YAML specification
	r.Get("/docs/swagger.yaml", SwaggerYAMLHandler())
}
