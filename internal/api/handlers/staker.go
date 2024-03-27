package handlers

import (
	"net/http"

	"github.com/babylonchain/staking-api-service/internal/services"
	"github.com/babylonchain/staking-api-service/internal/types"
)

// GetStakerDelegations @Summary Get staker delegations
// @Description Retrieves delegations for a given staker
// @Accept json
// @Produce json
// @Param staker_btc_pk query string true "Staker BTC Public Key"
// @Success 200 {object} PublicResponse[[]services.DelegationPublic]{array} "List of delegations and pagination token"
// @Failure 400 {object} types.Error "Error: Bad Request"
// @Router /staker/delegations [get]
func (h *Handler) GetStakerDelegations(request *http.Request) (*Result, *types.Error) {
	stakerBtcPk := request.URL.Query().Get("staker_btc_pk")
	if stakerBtcPk == "" {
		return nil, types.NewErrorWithMsg(http.StatusBadRequest, types.BadRequest, "staker_btc_pk is required")
	}

	delegations, paginationToken, err := h.services.DelegationsByStakerPk(request.Context(), stakerBtcPk, "")
	if err != nil {
		return nil, err
	}

	return NewResultWithPagination[[]services.DelegationPublic](delegations, paginationToken), nil
}
