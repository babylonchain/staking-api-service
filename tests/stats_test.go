package tests

import (
	"testing"
	"time"

	"github.com/babylonchain/staking-api-service/internal/db/model"
	"github.com/babylonchain/staking-api-service/internal/types"
	"github.com/babylonchain/staking-queue-client/client"
	"github.com/stretchr/testify/assert"
)

func TestOverallStatsShouldBeShardedInDb(t *testing.T) {
	activeStakingEvent := buildActiveStakingEvent(mockStakerHash, 10)
	// build the expired staking event based on the active staking event
	var expiredEvents []client.ExpiredStakingEvent
	for _, event := range activeStakingEvent {
		expiredEvents = append(expiredEvents, client.NewExpiredStakingEvent(event.StakingTxHashHex, types.ActiveTxType.ToString()))
	}
	testServer := setupTestServer(t, nil)
	defer testServer.Close()
	sendTestMessage(testServer.Queues.ActiveStakingQueueClient, activeStakingEvent)
	time.Sleep(2 * time.Second)
	sendTestMessage(testServer.Queues.ExpiredStakingQueueClient, expiredEvents)
	time.Sleep(5 * time.Second)

	// directly read from the db to check that we have more than 2 records in the overall stats collection
	results, err := inspectDbDocuments[model.OverallStatsDocument](t, model.OverallStatsCollection)
	if err != nil {
		t.Fatalf("Failed to inspect DB documents: %v", err)
	}
	assert.Equal(t, 2, len(results), "expected 2 logical shards in the overall stats collection")

	// Sum it up, we shall get 0 active tvl and 0 active delegations. the total should remain positive number

	var totalActiveTvl int64
	var totalActiveDelegations int64
	var totalTvl int64
	var totalDelegations int64
	for _, r := range results {
		totalActiveTvl += r.ActiveTvl
		totalActiveDelegations += r.ActiveDelegations
		totalTvl += r.TotalTvl
		totalDelegations += r.TotalDelegations
	}
	assert.Equal(t, int64(0), totalActiveTvl, "total acvtive tvl shall be 0 as all staking tx are now unbonded")
	assert.Equal(t, int64(0), totalActiveDelegations)
	assert.NotZero(t, totalTvl)
	assert.Equal(t, int64(10), totalDelegations)
}
