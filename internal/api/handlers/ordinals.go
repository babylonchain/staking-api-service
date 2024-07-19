package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/babylonchain/staking-api-service/internal/types"
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
		if utxo.Txid == "" || utxo.Vout < 0 {
			return nil, types.NewErrorWithMsg(http.StatusBadRequest, types.BadRequest, "invalid UTXO entry")
		}
	}
	return utxos, nil
}

func (h *Handler) VerifyUTXOs(request *http.Request) (*Result, *types.Error) {
	utxos, err := parseUTXORequestPayload(request, h.config.Ordinals.MaxUTXOs)
	if err != nil {
		errDetails := []types.ErrorDetail{
			{
				Message:   err.Err.Error(),
				Status:    err.StatusCode,
				ErrorCode: string(err.ErrorCode),
			},
		}
		response := types.SafeUTXOResponse{
			Error: errDetails,
		}
		return &Result{
			Status: http.StatusBadRequest,
			Data:   response,
		}, nil
	}

	results, errDetails := h.clients.Ordinals.VerifyUTXOs(request.Context(), utxos)
	response := types.SafeUTXOResponse{
		Data:  results,
		Error: errDetails,
	}

	return &Result{
		Status: http.StatusOK,
		Data:   response,
	}, nil
}
