package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/babylonchain/staking-api-service/internal/types"
)

func (s *Services) VerifyUTXOs(ctx context.Context, utxos []types.UTXORequest) ([]types.SafeUTXO, []types.ErrorDetail) {
	var results []types.SafeUTXO
	var errDetails []types.ErrorDetail

	for _, utxo := range utxos {
		url := fmt.Sprintf("%s/output/%s:%d", s.cfg.External.OrdinalAPIURL, utxo.Txid, utxo.Vout)

		// Create a new HTTP request
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			errDetails = append(errDetails, types.ErrorDetail{
				TxId:      utxo.Txid,
				Message:   "Failed to create HTTP request.",
				Status:    http.StatusInternalServerError,
				ErrorCode: "REQUEST_CREATION_ERROR",
			})
			continue
		}

		// Set the Accept header to application/json
		req.Header.Set("Accept", "application/json")

		// Perform the HTTP request
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil || resp.StatusCode != http.StatusOK {
			errDetails = append(errDetails, types.ErrorDetail{
				TxId:      utxo.Txid,
				Message:   "UTXO not found.",
				Status:    http.StatusNotFound,
				ErrorCode: "UTXO_NOT_FOUND",
			})
			continue
		}
		defer resp.Body.Close()

		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			errDetails = append(errDetails, types.ErrorDetail{
				TxId:      utxo.Txid,
				Message:   "Failed to read Ordinal API response.",
				Status:    http.StatusInternalServerError,
				ErrorCode: "READ_ERROR",
			})
			continue
		}

		var output types.OrdinalOutputResponse
		if err := json.Unmarshal(bodyBytes, &output); err != nil {
			errDetails = append(errDetails, types.ErrorDetail{
				TxId:      utxo.Txid,
				Message:   "Failed to decode Ordinal API response.",
				Status:    http.StatusInternalServerError,
				ErrorCode: "DECODE_ERROR",
			})
			continue
		}

		// Decode the runes field if it's not empty
		var runes []string
		if len(output.Runes) > 0 {
			if string(output.Runes) != "{}" {
				if err := json.Unmarshal(output.Runes, &runes); err != nil {
					errDetails = append(errDetails, types.ErrorDetail{
						TxId:      utxo.Txid,
						Message:   "Failed to decode runes field.",
						Status:    http.StatusInternalServerError,
						ErrorCode: "DECODE_RUNES_ERROR",
					})
					continue
				}
			}
		}

		safe := len(output.Inscriptions) == 0 && len(runes) == 0
		results = append(results, types.SafeUTXO{
			TxId: utxo.Txid,
			Brc20: !safe,
		})
	}

	return results, errDetails
}