package unisat

import (
	"context"
	"net/http"

	"github.com/babylonchain/staking-api-service/internal/types"
)

type UnisatClientInterface interface {
	GetBaseURL() string
	GetDefaultRequestTimeout() int
	GetHttpClient() *http.Client
	FetchInscriptionsUtxosByAddress(ctx context.Context, address string, cursor uint32) ([]*UnisatUTXO, *types.Error)
}
