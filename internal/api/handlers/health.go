package handlers

import (
	"net/http"
)

func (h *Handler) HealthCheck(request *http.Request) (*Result, *ApiError) {
	err := h.dbclient.Ping(request.Context())
	if err != nil {
		return nil, NewInternalServiceError(err)
	}

	return NewResult("Server is up and running"), nil
}
