package config

import (
	"errors"
)

type ClientsConfig struct {
	Timeout int `mapstructure:"timeout"`
}

func (cfg *ClientsConfig) Validate() error {
	if cfg.Timeout <= 0 {
		return errors.New("timeout cannot be smaller or equal to 0")
	}

	return nil
}
