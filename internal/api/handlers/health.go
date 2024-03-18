package handlers

import (
	"net/http"
)

func (h *Handler) HealthCheck(request *http.Request) (*Result, error) {
	return NewResult("Server is up and running"), nil
}
