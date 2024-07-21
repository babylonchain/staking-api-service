package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/babylonchain/staking-api-service/internal/types"
	"github.com/babylonchain/staking-api-service/internal/utils"
)

func parseUTXORequestPayload(request *http.Request, maxUTXOs int) ([]types.UTXORequest, *types.Error) {
	var utxos []types.UTXORequest
	if err := json.NewDecoder(request.Body).Decode(&utxos); err != nil {
		return nil, types.NewErrorWithMsg(http.StatusBadRequest, types.BadRequest, "invalid input format")
	}

	if len(utxos) == 0 {
		return nil, types.NewErrorWithMsg(http.StatusBadRequest, types.BadRequest, "empty UTXO array")
	}

	if len(utxos) > maxUTXOs {
		return nil, types.NewErrorWithMsg(http.StatusBadRequest, types.BadRequest, "too many UTXOs in the request")
	}

	for _, utxo := range utxos {
		if !utils.IsValidTxHash(utxo.Txid) {
			return nil, types.NewErrorWithMsg(http.StatusBadRequest, types.BadRequest, "invalid UTXO txid")
		} else if utxo.Vout < 0 {
			return nil, types.NewErrorWithMsg(http.StatusBadRequest, types.BadRequest, "invalid UTXO vout")
		}
	}
	return utxos, nil
}

func (h *Handler) VerifyUTXOs(request *http.Request) (*Result, *types.Error) {
	utxos, err := parseUTXORequestPayload(request, h.config.Ordinals.MaxUTXOs)
	if err != nil {
		return nil, err
	}
	results, err := h.services.VerifyUTXOs(request.Context(), utxos)
	if err != nil {
		return nil, err
	}

	return NewResult(results), nil
}
