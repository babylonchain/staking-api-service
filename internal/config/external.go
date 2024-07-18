package config

import (
	"errors"
	"net/url"
)

type ExternalConfig struct {
	OrdinalAPIURL string `mapstructure:"ordinal_api_url"`
}

func (cfg *ExternalConfig) Validate() error {
	if cfg.OrdinalAPIURL == "" {
		return errors.New("ordinal_api_url cannot be empty")
	}

	parsedURL, err := url.ParseRequestURI(cfg.OrdinalAPIURL)
	if err != nil {
		return errors.New("invalid ordinal_api_url")
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return errors.New("ordinal_api_url must start with http or https")
	}

	return nil
}