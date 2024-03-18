package api

import "github.com/go-chi/chi"

func (a *Server) SetupRoutes(r *chi.Mux) {
	handlers := a.handlers
	r.Get("/healthcheck", registerHandler(handlers.HealthCheck))
}
