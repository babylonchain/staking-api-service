package config

import (
	"fmt"
)

type QueueConfig struct {
	Region                string `mapstructure:"region"`
	ActiveStakingQueueUrl string `mapstructure:"active_staking_queue_url"`
}

func (cfg *QueueConfig) Validate() error {
	if cfg.Region == "" {
		return fmt.Errorf("missing queue region")
	}

	if cfg.ActiveStakingQueueUrl == "" {
		return fmt.Errorf("missing active staking queue URL")
	}
	return nil
}
