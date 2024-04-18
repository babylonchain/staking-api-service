package handlers

import (
	"net/http"

	"github.com/babylonchain/staking-api-service/internal/types"
)

// GetOverallStats gets overall stats for babylon staking
// @Summary Get Overall Stats
// @Description Fetches overall stats for babylon staking including tvl, total delegations, active tvl, active delegations and total stakers.
// @Produce json
// @Success 200 {object} PublicResponse[services.StatsPublic] "Overall stats for babylon staking"
// @Router /v1/stats [get]
func (h *Handler) GetOverallStats(request *http.Request) (*Result, *types.Error) {
	stats, err := h.services.GetOverallStats(request.Context())
	if err != nil {
		return nil, err
	}

	return NewResult(stats), nil
}
