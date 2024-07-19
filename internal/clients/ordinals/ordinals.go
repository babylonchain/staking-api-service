package ordinals

import (
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

func (c *OrdinalsClient) FetchUTXOInfo(ctx context.Context, txid string, vout int) (*types.OrdinalsOutputResponse, error) {
	url := fmt.Sprintf("%s:%s/output/%s:%d", c.config.Host, c.config.Port, txid, vout)

	// Set a timeout for the request
	ctx, cancel := context.WithTimeout(ctx, time.Duration(c.config.Timeout)*time.Millisecond)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)

	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to perform HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("UTXO not found.")
	}

	var output types.OrdinalsOutputResponse
	if err := json.NewDecoder(resp.Body).Decode(&output); err != nil {
		return nil, fmt.Errorf("failed to decode Ordinal API response: %w", err)
	}

	return &output, nil
}
