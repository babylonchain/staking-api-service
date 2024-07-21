package ordinals

import (
	"context"
	"fmt"
	"net/http"

	baseclient "github.com/babylonchain/staking-api-service/internal/clients/base"
	"github.com/babylonchain/staking-api-service/internal/config"
	"github.com/babylonchain/staking-api-service/internal/types"
)

type OrdinalsClient struct {
	config         *config.OrdinalsConfig
	defaultHeaders map[string]string
	httpClient     *http.Client
}

func NewOrdinalsClient(config *config.OrdinalsConfig) *OrdinalsClient {
	// Client is disabled if config is nil
	if config == nil {
		return nil
	}
	httpClient := &http.Client{}
	headers := map[string]string{
		"Content-Type": "application/json",
		"Accept":       "application/json",
	}
	return &OrdinalsClient{
		config,
		headers,
		httpClient,
	}
}

// Necessary for the BaseClient interface
func (c *OrdinalsClient) GetBaseURL() string {
	return fmt.Sprintf("%s:%s", c.config.Host, c.config.Port)
}

func (c *OrdinalsClient) GetDefaultRequestTimeout() int {
	return c.config.Timeout
}

func (c *OrdinalsClient) GetHttpClient() *http.Client {
	return c.httpClient
}

func (c *OrdinalsClient) FetchUTXOInfos(
	ctx context.Context, utxos []types.UTXORequest,
) ([]*types.OrdinalsOutputResponse, *types.Error) {
	path := "/outputs"

	opts := &baseclient.BaseClientOptions{
		Path:    path,
		Headers: c.defaultHeaders,
	}

	var txHashVouts []string
	for _, utxo := range utxos {
		txHashVouts = append(txHashVouts, fmt.Sprintf("%s:%d", utxo.Txid, utxo.Vout))
	}

	outputsResponse, err := baseclient.SendRequest[[]string, []types.OrdinalsOutputResponse](
		ctx, c, http.MethodPost, opts, &txHashVouts,
	)
	if err != nil {
		return nil, err
	}

	// convert the response to a map for easier lookup
	outputsMap := make(map[string]types.OrdinalsOutputResponse)
	for _, output := range *outputsResponse {
		outputsMap[output.Transaction] = output
	}

	// re-order the response based on the request order
	var outputs = make([]*types.OrdinalsOutputResponse, len(utxos))
	for i, utxo := range utxos {
		output, ok := outputsMap[utxo.Txid]
		if !ok {
			return nil, types.NewErrorWithMsg(
				http.StatusInternalServerError,
				types.InternalServiceError,
				"response does not contain all requested UTXOs",
			)
		}
		outputs[i] = &output
	}

	return outputs, nil
}
