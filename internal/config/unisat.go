package config

import (
	"errors"
	"net/url"
)

type UnisatConfig struct {
	Host     string `mapstructure:"host"`
	Timeout  int    `mapstructure:"timeout"`
	Limit    uint32 `mapstructure:"limit"`
	ApiToken string `mapstructure:"token"`
}

func (cfg *UnisatConfig) Validate() error {
	if cfg.Host == "" {
		return errors.New("host cannot be empty")
	}

	if cfg.Timeout <= 0 {
		return errors.New("timeout cannot be smaller or equal to 0")
	}

	if cfg.Limit <= 0 {
		return errors.New("limit cannot be smaller or equal to 0")
	}

	if cfg.ApiToken == "" {
		return errors.New("api token cannot be empty")
	}

	parsedURL, err := url.ParseRequestURI(cfg.Host)
	if err != nil {
		return errors.New("invalid unisat service host")
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return errors.New("host must start with http or https")
	}

	return nil
}
