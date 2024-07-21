package config

import (
	"errors"
	"net/url"
)

type OrdinalsConfig struct {
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	Timeout  int    `mapstructure:"timeout"`
	MaxUTXOs int    `mapstructure:"max_utxos"`
}

func (cfg *OrdinalsConfig) Validate() error {
	if cfg.Host == "" {
		return errors.New("host cannot be empty")
	}

	if cfg.Port == "" {
		return errors.New("port cannot be empty")
	}

	if cfg.Timeout <= 0 {
		return errors.New("timeout cannot be smaller or equal to 0")
	}

	if cfg.MaxUTXOs <= 0 {
		return errors.New("max_utxos cannot be smaller or equal to 0")
	}

	parsedURL, err := url.ParseRequestURI(cfg.Host)
	if err != nil {
		return errors.New("invalid ordinal service host")
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return errors.New("host must start with http or https")
	}

	return nil
}
