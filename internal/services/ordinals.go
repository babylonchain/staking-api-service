package services

import (
	"context"
	"net/http"

	"github.com/babylonchain/staking-api-service/internal/types"
)

func (s *Services) VerifyUTXOs(ctx context.Context, utxos []types.UTXORequest) ([]types.SafeUTXO, []types.ErrorDetail) {
	var results []types.SafeUTXO
	var errDetails []types.ErrorDetail

	for _, utxo := range utxos {
		output, err := s.Clients.Ordinals.FetchUTXOInfo(utxo.Txid, utxo.Vout)
		if err != nil {
			errDetails = append(errDetails, types.ErrorDetail{
				TxId:      utxo.Txid,
				Message:   err.Error(),
				Status:    http.StatusNotFound,
				ErrorCode: "UTXO_NOT_FOUND",
			})
			continue
		}

		safe := len(output.Inscriptions) == 0 && len(output.Runes) == 0
		results = append(results, types.SafeUTXO{
			TxId:        utxo.Txid,
			Inscription: !safe,
		})
	}

	return results, errDetails
}