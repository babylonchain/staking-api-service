package config

import (
	"fmt"
	"net/url"
	"strconv"
)

const (
	maxLogicalShardCount = 100
)

type DbConfig struct {
	DbName             string `mapstructure:"db-name"`
	Address            string `mapstructure:"address"`
	MaxPaginationLimit int64  `mapstructure:"max-pagination-limit"`
	DbBatchSizeLimit   int64  `mapstructure:"db-batch-size-limit"`
	LogicalShardCount  int64  `mapstructure:"logical-shard-count"`
}

func (cfg *DbConfig) Validate() error {
	if cfg.Address == "" {
		return fmt.Errorf("missing db address")
	}

	if cfg.DbName == "" {
		return fmt.Errorf("missing db name")
	}

	u, err := url.Parse(cfg.Address)
	if err != nil {
		return fmt.Errorf("invalid db address: %w", err)
	}

	if u.Scheme != "mongodb" {
		return fmt.Errorf("unsupported db scheme: %s", u.Scheme)
	}

	if u.Host == "" {
		return fmt.Errorf("missing host in db address")
	}

	port := u.Port()
	if port == "" {
		return fmt.Errorf("missing port in db address")
	}

	portNum, err := strconv.Atoi(port)
	if err != nil {
		return fmt.Errorf("invalid port in db address: %w", err)
	}

	if portNum < 1024 || portNum > 65535 {
		return fmt.Errorf("port number must be between 1024 and 65535 (inclusive)")
	}

	if cfg.MaxPaginationLimit <= 0 {
		return fmt.Errorf("max pagination limit must be greater than 0")
	}

	if cfg.DbBatchSizeLimit <= 0 {
		return fmt.Errorf("db batch size limit must be greater than 0")
	}

	if cfg.LogicalShardCount <= 1 {
		return fmt.Errorf("logical shard count must be greater than 1")
	}

	// Below is adding as a safety net to avoid performance issue.
	// Changes to the logical shard count shall be discussed with the team
	if cfg.LogicalShardCount > maxLogicalShardCount {
		return fmt.Errorf("large logical shard count will have significant performance impact, please inform the team before changing this value")
	}

	return nil
}
