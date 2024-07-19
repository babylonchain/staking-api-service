package ordinals

import (
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
	httpClient := &http.Client{
		Timeout: time.Duration(config.Timeout),
	}
	return &OrdinalsClient{
		config,
		httpClient,
	}
}

func (c *OrdinalsClient) FetchUTXOInfo(txid string, vout int) (*types.OrdinalOutputResponse, error) {
	url := fmt.Sprintf("%s:%s/output/%s:%d", c.config.Host, c.config.Port, txid, vout)

	req, err := http.NewRequest("GET", url, nil)

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

	var output types.OrdinalOutputResponse
	if err := json.NewDecoder(resp.Body).Decode(&output); err != nil {
		return nil, fmt.Errorf("failed to decode Ordinal API response: %w", err)
	}

	return &output, nil
}
