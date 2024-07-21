package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/babylonchain/staking-api-service/internal/types"
	"github.com/babylonchain/staking-api-service/internal/utils"
	"github.com/btcsuite/btcd/chaincfg"
)

type VerifyUTXOsRequestPayload struct {
	Address string              `json:"address"`
	Utxos   []types.UTXORequest `json:"utxos"`
}

func parseRequestPayload(request *http.Request, maxUTXOs int, netParam *chaincfg.Params) (*VerifyUTXOsRequestPayload, *types.Error) {
	var payload VerifyUTXOsRequestPayload
	if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
		return nil, types.NewErrorWithMsg(http.StatusBadRequest, types.BadRequest, "invalid input format")
	}
	utxos := payload.Utxos
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

	err := utils.IsValidBtcAddress(payload.Address, netParam)
	if err != nil {
		return nil, types.NewErrorWithMsg(http.StatusBadRequest, types.BadRequest, err.Error())
	}
	return &payload, nil
}

func (h *Handler) VerifyUTXOs(request *http.Request) (*Result, *types.Error) {
	inputs, err := parseRequestPayload(request, h.config.Ordinals.MaxUTXOs, h.config.Server.BTCNetParam)
	if err != nil {
		return nil, err
	}

	results, err := h.services.VerifyUTXOs(request.Context(), inputs.Utxos, inputs.Address)
	if err != nil {
		return nil, err
	}

	return NewResult(results), nil
}
