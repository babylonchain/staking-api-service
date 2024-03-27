package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/babylonchain/staking-api-service/internal/types"
)

type UnbondDelegationRequestPayload struct {
	UnbondingTxHashHex       string `json:"unbonding_tx_hash_hex"`
	UnbondingTxHex           string `json:"unbonding_tx_hex"`
	StakerSignedSignatureHex string `json:"staker_signed_signature_hex"`
}

func (h *Handler) UnbondDelegation(request *http.Request) (*Result, *types.Error) {
	payload := &UnbondDelegationRequestPayload{}
	err := json.NewDecoder(request.Body).Decode(payload)
	if err != nil {
		return nil, types.NewErrorWithMsg(http.StatusBadRequest, types.BadRequest, "invalid request payload")
	}

	return nil, nil
}
