package handlers

import (
	"fmt"
	"net/http"

	"github.com/babylonchain/staking-api-service/internal/types"
)

// GetOverallStats gets overall stats for babylon staking
// @Summary Get Overall Stats
// @Description Fetches overall stats for babylon staking including tvl, total delegations, active tvl, active delegations and total stakers.
// @Produce json
// @Success 200 {object} PublicResponse[services.OverallStatsPublic] "Overall stats for babylon staking"
// @Router /v1/stats [get]
func (h *Handler) GetOverallStats(request *http.Request) (*Result, *types.Error) {
	stats, err := h.services.GetOverallStats(request.Context())
	if err != nil {
		return nil, err
	}

	return NewResult(stats), nil
}

// GetTopStakerStats gets top stakers by active tvl
// @Summary Get Top Staker Stats by Active TVL
// @Description Fetches details of top stakers by their active total value locked (ActiveTvl) in descending order.
// @Produce json
// @Param  pagination_key query string false "Pagination key to fetch the next page of top stakers"
// @Success 200 {object} PublicResponse[[]services.StakerStatsPublic]{array} "List of top stakers by active tvl"
// @Failure 400 {object} types.Error "Error: Bad Request"
// @Router /v1/stats/staker [get]
func (h *Handler) GetTopStakerStats(request *http.Request) (*Result, *types.Error) {
	paginationKey, err := parsePaginationQuery(request)
	if err != nil {
		return nil, err
	}
	topStakerStats, paginationToken, err := h.services.GetTopStakersByActiveTvl(request.Context(), paginationKey)
	if err != nil {
		return nil, err
	}

	return NewResultWithPagination(topStakerStats, paginationToken), nil
}

// GetStakerStats gets stats for a specific staker
// @Summary Get Staker Stats
// @Description Fetches stats for a specific staker including tvl, total delegations, active tvl, active delegations, and withdrawable tvl.
// @Produce json
// @Param staker_pk_hex query string true "user public key hex"
// @Success 200 {object} PublicResponse[services.StakerStatsPublic] "Stats for the specific staker"
// @Failure 400 {object} types.Error "Error: Bad Request"
// @Router /v1/stats/single-staker [get]
func (h *Handler) GetStakerStats(request *http.Request) (*Result, *types.Error) {
	stakerPkHex := request.URL.Query().Get("staker_pk_hex")
	if stakerPkHex == "" {
		return nil, &types.Error{
			StatusCode:   http.StatusBadRequest,
			ErrorCode:    types.BadRequest,
			Err:          fmt.Errorf("staker_pk_hex query parameter is required"),
		}
	}

	stats, err := h.services.GetStakerStats(request.Context(), stakerPkHex)
	if err != nil {
		return nil, err
	}

	return NewResult(stats), nil
}