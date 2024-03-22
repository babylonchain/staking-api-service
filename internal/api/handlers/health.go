package handlers

import (
	"net/http"

	"github.com/babylonchain/staking-api-service/internal/types"
)

func (h *Handler) HealthCheck(request *http.Request) (*Result, *types.Error) {
	err := h.services.DoHealthCheck(request.Context())
	if err != nil {
		return nil, types.NewInternalServiceError(err)
	}

	return NewResult("Server is up and running"), nil
}
