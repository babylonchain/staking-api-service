package clients

import (
	"github.com/babylonchain/staking-api-service/internal/clients/ordinals"
	"github.com/babylonchain/staking-api-service/internal/config"
)

type Clients struct {
	Ordinals *ordinals.OrdinalsClient
}

func New(cfg *config.Config) *Clients {
	ordinalsClient := ordinals.NewOrdinalsClient(&config.OrdinalsConfig{
		Host:     cfg.Ordinals.Host,
		Port:     cfg.Ordinals.Port,
		Timeout:  cfg.Ordinals.Timeout,
		MaxUTXOs: cfg.Ordinals.MaxUTXOs,
	})

	return &Clients{
		Ordinals: ordinalsClient,
	}
}
