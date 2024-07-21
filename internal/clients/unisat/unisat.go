package unisat

import (
	"context"
	"fmt"
	"net/http"

	baseclient "github.com/babylonchain/staking-api-service/internal/clients/base"
	"github.com/babylonchain/staking-api-service/internal/config"
	"github.com/babylonchain/staking-api-service/internal/types"
)

type UnisatInscriptions struct {
	InscriptionId     string `json:"inscriptionId"`
	InscriptionNumber int    `json:"inscriptionNumber"`
	IsBRC20           bool   `json:"isBRC20"`
	Moved             bool   `json:"moved"`
	Offset            int    `json:"offset"`
}

type UnisatUtxos struct {
	TxId         string                `json:"txid"`
	Vout         int                   `json:"vout"`
	Inscriptions []*UnisatInscriptions `json:"inscriptions"`
}

type UnisatResponseData struct {
	Cursor int            `json:"cursor"`
	Total  int            `json:"total"`
	Utxo   []*UnisatUtxos `json:"utxo"`
}

// Refer to https://open-api.unisat.io/swagger.html
type UnisatResponse struct {
	Code int                `json:"code"`
	Msg  string             `json:"msg"`
	Data UnisatResponseData `json:"data"`
}

type UnisatClient struct {
	config        *config.UnisatConfig
	httpClient    *http.Client
	defaultHeader map[string]string
}

func NewUnisatClient(config *config.UnisatConfig) *UnisatClient {
	// Client is disabled if config is nil
	if config == nil {
		return nil
	}
	httpClient := &http.Client{}
	defaultHeader := map[string]string{
		"Accept":        "application/json",
		"Authorization": fmt.Sprintf("Bearer %s", config.ApiToken),
	}
	return &UnisatClient{
		config,
		httpClient,
		defaultHeader,
	}
}

// Necessary for the BaseClient interface
func (c *UnisatClient) GetBaseURL() string {
	return fmt.Sprintf("%s", c.config.Host)
}

func (c *UnisatClient) GetDefaultRequestTimeout() int {
	return c.config.Timeout
}

func (c *UnisatClient) GetHttpClient() *http.Client {
	return c.httpClient
}

// FetchInscriptionsUtxosByAddress fetches inscription UTXOs by address
// Refer to https://open-api.unisat.io/swagger.html#/address
// cursor and limit are used for pagination
func (c *UnisatClient) FetchInscriptionsUtxosByAddress(
	ctx context.Context, address string, cursor int,
) ([]*UnisatUtxos, *types.Error) {
	path := fmt.Sprintf(
		"/v1/indexer/address/%s/inscription-utxo-data?cursor=%d&size=%d",
		address, cursor, c.config.Limit,
	)
	opts := &baseclient.BaseClientOptions{
		Path:    path,
		Headers: c.defaultHeader,
	}

	resp, err := baseclient.SendRequest[any, UnisatResponse](
		ctx, c, http.MethodGet, opts, nil,
	)
	if err != nil {
		return nil, err
	}

	return resp.Data.Utxo, nil
}
