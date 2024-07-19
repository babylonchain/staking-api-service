package clients

import (
	"net/http"
	"time"

	"github.com/babylonchain/staking-api-service/internal/clients/ordinals"
	"github.com/babylonchain/staking-api-service/internal/config"
)

type Clients struct {
	Ordinals *ordinals.OrdinalsClient
}

func New(cfg *config.Config) *Clients {
	httpClient := &http.Client{
		Timeout: time.Duration(cfg.Clients.Timeout) * time.Second,
	}

	ordinalsClient := ordinals.NewOrdinalsClient(&config.OrdinalsConfig{
		Host:     cfg.Ordinals.Host,
		Port:     cfg.Ordinals.Port,
		MaxUTXOs: cfg.Ordinals.MaxUTXOs,
	}, httpClient)

	return &Clients{
		Ordinals: ordinalsClient,
	}
}
