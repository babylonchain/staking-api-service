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
	/*
		FetchInscriptionsUtxosByAddress fetches inscription UTXOs by address
		Refer to https://open-api.unisat.io/swagger.html#/address
		cursor and limit are used for pagination
	*/
	FetchInscriptionsUtxosByAddress(ctx context.Context, address string, cursor uint32) ([]*UnisatUTXO, *types.Error)
}
