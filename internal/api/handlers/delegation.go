package handlers

import (
	"net/http"

	"github.com/babylonchain/staking-api-service/internal/types"
)

// GetDelegationByTxHash @Summary Get a delegation
// @Description Retrieves a delegation by a given transaction hash
// @Produce json
// @Param tx_hash query string true "Transaction Hash"
// @Success 200 {object} PublicResponse[services.DelegationPublic] "Delegation"
// @Failure 400 {object} types.Error "Error: Bad Request"
// @Router /v1/delegation/ [get]
func (h *Handler) GetDelegationByTxHash(request *http.Request) (*Result, *types.Error) {
	txHash := request.URL.Query().Get("tx_hash")
	if txHash == "" {
		return nil, types.NewErrorWithMsg(http.StatusBadRequest, types.BadRequest, "tx_hash is required")
	}

	delegation, err := h.services.GetDelegation(request.Context(), txHash)
	if err != nil {
		return nil, err
	}

	return NewResult(delegation), nil
}
