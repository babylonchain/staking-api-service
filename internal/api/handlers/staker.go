package handlers

import (
	"net/http"

	"github.com/babylonchain/staking-api-service/internal/types"
)

// GetStakerDelegations @Summary Get staker delegations
// @Description Retrieves delegations for a given staker
// @Produce json
// @Param staker_btc_pk query string true "Staker BTC Public Key"
// @Param pagination_key query string false "Pagination key to fetch the next page of delegations"
// @Success 200 {object} PublicResponse[[]services.DelegationPublic]{array} "List of delegations and pagination token"
// @Failure 400 {object} types.Error "Error: Bad Request"
// @Router /v1/staker/delegations [get]
func (h *Handler) GetStakerDelegations(request *http.Request) (*Result, *types.Error) {
	stakerBtcPk := request.URL.Query().Get("staker_btc_pk")
	if stakerBtcPk == "" {
		return nil, types.NewErrorWithMsg(http.StatusBadRequest, types.BadRequest, "staker_btc_pk is required")
	}

	paginationKey := request.URL.Query().Get("pagination_key")

	delegations, newPaginationKey, err := h.services.DelegationsByStakerPk(request.Context(), stakerBtcPk, paginationKey)
	if err != nil {
		return nil, err
	}

	return NewResultWithPagination(delegations, newPaginationKey), nil
}
