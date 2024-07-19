package ordinals

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/babylonchain/staking-api-service/internal/config"
	"github.com/babylonchain/staking-api-service/internal/types"
)

type OrdinalsClient struct {
	config     *config.OrdinalsConfig
	httpClient *http.Client
}

func NewOrdinalsClient(config *config.OrdinalsConfig) *OrdinalsClient {
	httpClient := &http.Client{}
	return &OrdinalsClient{
		config,
		httpClient,
	}
}

func (c *OrdinalsClient) FetchUTXOInfos(ctx context.Context, utxos []types.UTXORequest) ([]types.OrdinalsOutputResponse, *types.Error) {
	url := fmt.Sprintf("%s:%s/outputs", c.config.Host, c.config.Port)

	// Set a timeout for the request
	ctx, cancel := context.WithTimeout(ctx, time.Duration(c.config.Timeout)*time.Millisecond)
	defer cancel()

	var txHashVouts []string
	for _, utxo := range utxos {
		txHashVouts = append(txHashVouts, fmt.Sprintf("%s:%d", utxo.Txid, utxo.Vout))
	}

	body, err := json.Marshal(txHashVouts)
	if err != nil {
		return nil, types.NewErrorWithMsg(http.StatusInternalServerError, types.InternalServiceError, "failed to marshal request body")
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, types.NewErrorWithMsg(http.StatusInternalServerError, types.InternalServiceError, fmt.Sprintf("failed to create HTTP request: %v", err))
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)

	if err != nil {
		return nil, types.NewErrorWithMsg(http.StatusInternalServerError, types.InternalServiceError, fmt.Sprintf("failed to perform HTTP request: %v", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnprocessableEntity {
		return nil, types.NewErrorWithMsg(http.StatusUnprocessableEntity, types.UnprocessableEntity, "unprocessable entity")
	} else if resp.StatusCode == http.StatusNotFound {
		return nil, types.NewErrorWithMsg(http.StatusNotFound, types.NotFound, "UTXOs not found")
	} else if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		return nil, types.NewErrorWithMsg(resp.StatusCode, types.BadRequest, "client error")
	} else if resp.StatusCode >= 500 {
		return nil, types.NewErrorWithMsg(resp.StatusCode, types.InternalServiceError, "server error")
	}

	var outputs []types.OrdinalsOutputResponse
	if err := json.NewDecoder(resp.Body).Decode(&outputs); err != nil {
		return nil, types.NewErrorWithMsg(http.StatusInternalServerError, types.InternalServiceError, fmt.Sprintf("failed to decode Ordinal API response: %v", err))
	}

	return outputs, nil
}
