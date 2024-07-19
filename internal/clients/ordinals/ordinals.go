package ordinals

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/babylonchain/staking-api-service/internal/config"
	"github.com/babylonchain/staking-api-service/internal/types"
)

type OrdinalsClient struct {
	config *config.OrdinalsConfig
	client *http.Client
}

func NewOrdinalsClient(config *config.OrdinalsConfig, httpClient *http.Client) *OrdinalsClient {
	return &OrdinalsClient{
		config: config,
		client: httpClient,
	}
}

func (c *OrdinalsClient) fetchUTXOInfo(txid string, vout int) (*types.OrdinalOutputResponse, error) {
	url := fmt.Sprintf("%s:%s/output/%s:%d", c.config.Host, c.config.Port, txid, vout)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to perform HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("UTXO not found.")
	}

	var output types.OrdinalOutputResponse
	if err := json.NewDecoder(resp.Body).Decode(&output); err != nil {
		return nil, fmt.Errorf("failed to decode Ordinal API response: %w", err)
	}

	return &output, nil
}

func (c *OrdinalsClient) VerifyUTXOs(ctx context.Context, utxos []types.UTXORequest) ([]types.SafeUTXO, []types.ErrorDetail) {
	var results []types.SafeUTXO
	var errDetails []types.ErrorDetail

	for _, utxo := range utxos {
		output, err := c.fetchUTXOInfo(utxo.Txid, utxo.Vout)
		if err != nil {
			errDetails = append(errDetails, types.ErrorDetail{
				TxId:      utxo.Txid,
				Message:   err.Error(),
				Status:    http.StatusNotFound,
				ErrorCode: "UTXO_NOT_FOUND",
			})
			continue
		}

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
			TxId:        utxo.Txid,
			Inscription: !safe,
		})
	}

	return results, errDetails
}
