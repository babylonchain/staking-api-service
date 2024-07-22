package clients

import (
	"github.com/babylonchain/staking-api-service/internal/clients/ordinals"
	"github.com/babylonchain/staking-api-service/internal/clients/unisat"
	"github.com/babylonchain/staking-api-service/internal/config"
)

type Clients struct {
	Ordinals *ordinals.OrdinalsClient
	Unisat   *unisat.UnisatClient
}

func New(cfg *config.Config) *Clients {
	ordinalsClient := ordinals.NewOrdinalsClient(cfg.Ordinals)
	unisatClient := unisat.NewUnisatClient(cfg.Unisat)

	return &Clients{
		Ordinals: ordinalsClient,
		Unisat:   unisatClient,
	}
}
