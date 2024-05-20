package db

import (
	"testing"

	"github.com/babylonchain/staking-api-service/internal/config"
	"go.mongodb.org/mongo-driver/mongo"
)

func TestTxWithRetries_ExponentialBackoff(t *testing.T) {
	// create a new db instance
	abc := &mongo.Client{}

	db := Database{
		DbName: "test",
		Client: nil,
		cfg: config.DbConfig{
			Address: "localhost",
			DbName:  "test",
		},
	}
}
