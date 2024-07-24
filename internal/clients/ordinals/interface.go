package ordinals

import (
	"context"
	"net/http"

	"github.com/babylonchain/staking-api-service/internal/types"
)

type OrdinalsClientInterface interface {
	GetBaseURL() string
	GetDefaultRequestTimeout() int
	GetHttpClient() *http.Client
	FetchUTXOInfos(ctx context.Context, utxos []types.UTXOIdentifier) ([]OrdinalsOutputResponse, *types.Error)
}
