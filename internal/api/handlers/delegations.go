package handlers

import (
	"net/http"

	"github.com/babylonchain/staking-api-service/internal/types"
)

func (h *Handler) GetStakerDelegations(request *http.Request) (*Result, *types.Error) {
	stakerBtcPk := request.URL.Query().Get("staker_btc_pk")
	if stakerBtcPk == "" {
		return nil, types.NewErrorWithMsg(http.StatusBadRequest, types.BadRequest, "staker_btc_pk is required")
	}

	delegations, paginationToken, err := h.services.DelegationsByStakerPk(request.Context(), stakerBtcPk, "")
	if err != nil {
		return nil, err
	}

	return NewResultWithPagination(delegations, paginationToken), nil
}
