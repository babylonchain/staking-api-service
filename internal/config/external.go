package config

import (
	"errors"
	"net/url"
)

type OrdinalsConfig struct {
	OrdinalsAPIHost string `mapstructure:"ordinals_api_host"`
	OrdinalsAPIPort string `mapstructure:"ordinals_api_port"`
	Timeout         int    `mapstructure:"timeout"`
	MaxUTXOs        int    `mapstructure:"max_utxos"`
}

func (cfg *OrdinalsConfig) Validate() error {
	if cfg.OrdinalsAPIHost == "" {
		return errors.New("ordinals_api_host cannot be empty")
	}

	if cfg.OrdinalsAPIPort == "" {
		return errors.New("ordinals_api_port cannot be empty")
	}

	if cfg.Timeout <= 0 {
		return errors.New("timeout cannot be smaller or equal to 0")
	}

	if cfg.MaxUTXOs < 0 {
		return errors.New("max_utxos cannot be smaller than 0")
	}

	parsedURL, err := url.ParseRequestURI(cfg.OrdinalsAPIHost)
	if err != nil {
		return errors.New("invalid ordinals_api_host")
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return errors.New("ordinals_api_host must start with http or https")
	}

	return nil
}
