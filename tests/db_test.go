package tests

import (
	"context"
	"testing"
	"time"

	"github.com/babylonchain/staking-api-service/internal/config"
	"github.com/babylonchain/staking-api-service/internal/db"
	"github.com/babylonchain/staking-api-service/internal/observability/metrics"
	"github.com/stretchr/testify/require"
)

func TestDbReconnect(t *testing.T) {
	cfg, err := config.New("./config/config-test.yml")
	if err != nil {
		t.Fatalf("Failed to load test config: %v", err)
	}
	metricsPort := cfg.Metrics.GetMetricsPort()
	metrics.Init(metricsPort)
	dbClient, err := db.New(context.TODO(), cfg.Db)
	require.NoError(t, err)
	require.NotNil(t, dbClient)

	dbClient.StartConnectionCheckRoutine(context.Background())
	err = dbClient.Ping(context.Background())
	require.NoError(t, err)

	// Close the connection
	closeErr := dbClient.Client.Disconnect(context.Background())
	require.NoError(t, closeErr)
	// There should be an error when pinging the database as the connection is closed
	err = dbClient.Ping(context.Background())
	require.Error(t, err)

	// Wait for the connection to be re-established
	time.Sleep(cfg.Db.ConnectionCheckPeriod)
	// Ping the database again, connection should be re-established
	err = dbClient.Ping(context.Background())
	require.NoError(t, err)
}
