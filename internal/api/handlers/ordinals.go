package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/babylonchain/staking-api-service/internal/types"
)

func parseUTXORequestPayload(request *http.Request) ([]types.UTXORequest, *types.Error) {
	var utxos []types.UTXORequest
	if err := json.NewDecoder(request.Body).Decode(&utxos); err != nil {
		return nil, types.NewErrorWithMsg(http.StatusBadRequest, types.BadRequest, "invalid input format")
	}
	
	if len(utxos) == 0 {
		return nil, types.NewErrorWithMsg(http.StatusBadRequest, types.BadRequest, "empty UTXO array")
	}

	for _, utxo := range utxos {
		if utxo.Txid == "" || utxo.Vout < 0 {
			return nil, types.NewErrorWithMsg(http.StatusBadRequest, types.BadRequest, "invalid UTXO entry")
		}
	}
	return utxos, nil
}

// VerifyUTXOs @Summary Verify UTXOs
// @Description Verifies if given UTXOs contain BRC-20 tokens
// @Accept json
// @Produce json
// @Param UTXO body []types.UTXORequest true "Array of UTXOs"
// @Success 200 {object} types.SafeUTXOResponse "Verification Results"
// @Failure 400 {object} types.Error "Error: Bad Request"
// @Failure 500 {object} types.Error "Error: Internal Server Error"
// @Router /v1/ordinals/verify-utxos [post]
func (h *Handler) VerifyUTXOs(request *http.Request) (*Result, *types.Error) {
	utxos, err := parseUTXORequestPayload(request)
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

	results, errDetails := h.services.VerifyUTXOs(request.Context(), utxos)
	response := types.SafeUTXOResponse{
		Data:  results,
		Error: errDetails,
	}

	return &Result{
		Status: http.StatusOK,
		Data:   response,
	}, nil
}