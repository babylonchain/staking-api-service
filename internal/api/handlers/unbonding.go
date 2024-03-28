package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/babylonchain/staking-api-service/internal/types"
)

type UnbondDelegationRequestPayload struct {
	StakingTxHashHex         string `json:"staking_tx_hash_hex"`
	UnbondingTxHashHex       string `json:"unbonding_tx_hash_hex"`
	UnbondingTxHex           string `json:"unbonding_tx_hex"`
	StakerSignedSignatureHex string `json:"staker_signed_signature_hex"`
}

// UnbondDelegation godoc
// @Summary Unbond delegation
// @Description Unbonds a delegation by processing the provided transaction details.
// @Tags unbonding
// @Accept json
// @Produce json
// @Param payload body UnbondDelegationRequestPayload true "Unbonding Request Payload"
// @Success 202 "Request accepted and will be processed asynchronously"
// @Failure 400 {object} types.Error "Invalid request payload"
// @Router /v1/unbonding [post]
func (h *Handler) UnbondDelegation(request *http.Request) (*Result, *types.Error) {
	payload := &UnbondDelegationRequestPayload{}
	err := json.NewDecoder(request.Body).Decode(payload)
	if err != nil {
		return nil, types.NewErrorWithMsg(http.StatusBadRequest, types.BadRequest, "invalid request payload")
	}
	unbondErr := h.services.UnbondDelegation(
		request.Context(), payload.StakingTxHashHex,
		payload.UnbondingTxHashHex, payload.UnbondingTxHex,
		payload.StakerSignedSignatureHex,
	)
	if unbondErr != nil {
		return nil, unbondErr
	}

	return &Result{Status: http.StatusAccepted}, nil
}

// GetUnbondingEligibility godoc
// @Summary Check unbonding eligibility
// @Description Checks if a delegation identified by its staking transaction hash is eligible for unbonding.
// @Tags unbonding
// @Accept json
// @Produce json
// @Param staking_tx_hash_hex query string true "Staking Transaction Hash Hex"
// @Success 200 "The delegation is eligible for unbonding"
// @Failure 400 {object} types.Error "Missing or invalid 'staking_tx_hash_hex' query parameter"
// @Router /v1/unbonding/eligibility [get]
func (h *Handler) GetUnbondingEligibility(request *http.Request) (*Result, *types.Error) {
	stakingTxHashHex := request.URL.Query().Get("staking_tx_hash_hex")
	if stakingTxHashHex == "" {
		return nil, types.NewErrorWithMsg(http.StatusBadRequest, types.BadRequest, "staking_tx_hash_hex is required")
	}

	err := h.services.IsEligibleForUnbonding(request.Context(), stakingTxHashHex)
	if err != nil {
		return nil, err
	}

	return &Result{Status: http.StatusOK}, nil
}
