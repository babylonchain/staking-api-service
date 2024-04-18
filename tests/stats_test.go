package tests

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/babylonchain/staking-api-service/internal/api/handlers"
	"github.com/babylonchain/staking-api-service/internal/db/model"
	"github.com/babylonchain/staking-api-service/internal/services"
	"github.com/babylonchain/staking-api-service/internal/types"
	"github.com/babylonchain/staking-queue-client/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	overallStatsEndpoint = "/v1/stats"
)

func TestStatsShouldBeShardedInDb(t *testing.T) {
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
	time.Sleep(2 * time.Second)

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

	// We also check the finality provider stats, make sure it's sharded as well
	shardedFinalityProviderStats, err := inspectDbDocuments[model.FinalityProviderStatsDocument](t, model.FinalityProviderStatsCollection)
	if err != nil {
		t.Fatalf("Failed to inspect DB documents: %v", err)
	}
	assert.Less(t, 10, len(shardedFinalityProviderStats), "we inserted 10 staking tx, we shall expect more than 10 in db as it's sharded")
}

func TestStatsCalculationShouldOnlyProcessActiveAndUnbondedEvents(t *testing.T) {
	activeStakingEvent := getTestActiveStakingEvent()
	testServer := setupTestServer(t, nil)
	defer testServer.Close()

	err := sendTestMessage(testServer.Queues.ActiveStakingQueueClient, []client.ActiveStakingEvent{activeStakingEvent})
	require.NoError(t, err)

	time.Sleep(2 * time.Second)

	// Let's make a POST request to the unbonding endpoint
	unbondingUrl := testServer.Server.URL + unbondingPath
	requestBody := getTestUnbondDelegationRequestPayload(activeStakingEvent.StakingTxHashHex)
	requestBodyBytes, err := json.Marshal(requestBody)
	assert.NoError(t, err, "marshalling request body should not fail")

	resp, err := http.Post(unbondingUrl, "application/json", bytes.NewReader(requestBodyBytes))
	assert.NoError(t, err, "making POST request to unbonding endpoint should not fail")
	defer resp.Body.Close()

	// Let's inspect what's stored in the database
	results, err := inspectDbDocuments[model.UnbondingDocument](t, model.UnbondingCollection)
	assert.NoError(t, err, "failed to inspect DB documents")

	assert.Equal(t, 1, len(results), "expected 1 document in the DB")
	assert.Equal(t, "INSERTED", results[0].State)
	assert.Equal(t, activeStakingEvent.StakingTxHex, results[0].StakingTxHex)

	// Let's send an unbonding event
	unbondingEvent := client.UnbondingStakingEvent{
		EventType:               client.UnbondingStakingEventType,
		StakingTxHashHex:        requestBody.StakingTxHashHex,
		UnbondingTxHashHex:      requestBody.UnbondingTxHashHex,
		UnbondingTxHex:          requestBody.UnbondingTxHex,
		UnbondingTimeLock:       10,
		UnbondingStartTimestamp: time.Now().Unix(),
		UnbondingStartHeight:    activeStakingEvent.StakingStartHeight + 100,
		UnbondingOutputIndex:    1,
	}

	sendTestMessage(testServer.Queues.UnbondingStakingQueueClient, []client.UnbondingStakingEvent{unbondingEvent})
	time.Sleep(2 * time.Second)

	// directly read from the db to check that we only have 1 shard in the overall stats collection
	stats, err := inspectDbDocuments[model.OverallStatsDocument](t, model.OverallStatsCollection)
	if err != nil {
		t.Fatalf("Failed to inspect DB documents: %v", err)
	}
	assert.Equal(t, 1, len(stats), "expected 1 logical shards in the overall stats collection")

	// The stats should equal to the active staking event. Unbonding event should not affect the stats
	assert.NotZero(t, stats[0].ActiveTvl)
	assert.Equal(t, int64(1), stats[0].ActiveDelegations)
	assert.NotZero(t, stats[0].TotalTvl)
	assert.Equal(t, int64(1), stats[0].TotalDelegations)
}

func TestStatsEndpoints(t *testing.T) {
	activeStakingEvent := getTestActiveStakingEvent()
	testServer := setupTestServer(t, nil)
	defer testServer.Close()
	sendTestMessage(testServer.Queues.ActiveStakingQueueClient, []client.ActiveStakingEvent{activeStakingEvent})
	time.Sleep(2 * time.Second)

	// Test the finality endpoint first
	result := fetchFinalityEndpoint(t, testServer)
	assert.Equal(t, int64(activeStakingEvent.StakingValue), result[0].ActiveTvl)
	assert.Equal(t, int64(activeStakingEvent.StakingValue), result[0].TotalTvl)
	assert.Equal(t, int64(1), result[0].ActiveDelegations)
	assert.Equal(t, int64(1), result[0].TotalDelegations)

	// Test the overall stats endpoint
	overallStats := fetchOverallStatsEndpoint(t, testServer)
	assert.Equal(t, int64(activeStakingEvent.StakingValue), overallStats.ActiveTvl)
	assert.Equal(t, int64(activeStakingEvent.StakingValue), overallStats.TotalTvl)
	assert.Equal(t, int64(1), overallStats.ActiveDelegations)
	assert.Equal(t, int64(1), overallStats.TotalDelegations)
	assert.Equal(t, int64(0), overallStats.TotalStakers, "Should return default of 0 for unique number of staker. Yet to be implemented")

	// Now let's send an expired timelock event, this will affect the active stats only
	expiredEvent := client.NewExpiredStakingEvent(activeStakingEvent.StakingTxHashHex, types.ActiveTxType.ToString())
	sendTestMessage(testServer.Queues.ExpiredStakingQueueClient, []client.ExpiredStakingEvent{expiredEvent})
	time.Sleep(2 * time.Second)

	// Make a GET request to the finality providers endpoint
	result = fetchFinalityEndpoint(t, testServer)
	assert.Equal(t, int64(0), result[0].ActiveTvl)
	assert.Equal(t, int64(activeStakingEvent.StakingValue), result[0].TotalTvl)
	assert.Equal(t, int64(0), result[0].ActiveDelegations)
	assert.Equal(t, int64(1), result[0].TotalDelegations)

	overallStats = fetchOverallStatsEndpoint(t, testServer)
	assert.Equal(t, int64(0), overallStats.ActiveTvl)
	assert.Equal(t, int64(activeStakingEvent.StakingValue), overallStats.TotalTvl)
	assert.Equal(t, int64(0), overallStats.ActiveDelegations)
	assert.Equal(t, int64(1), overallStats.TotalDelegations)
	assert.Equal(t, int64(0), overallStats.TotalStakers, "Should return default of 0 for unique number of staker. Yet to be implemented")

	// Send two new active events, it will increment the stats
	activeEvents := buildActiveStakingEvent(mockStakerHash, 2)
	sendTestMessage(testServer.Queues.ActiveStakingQueueClient, activeEvents)
	time.Sleep(2 * time.Second)

	overallStats = fetchOverallStatsEndpoint(t, testServer)
	expectedTvl := int64(activeEvents[0].StakingValue + activeEvents[1].StakingValue)
	expectedTotalTvl := int64(expectedTvl) + int64(activeStakingEvent.StakingValue)
	assert.Equal(t, expectedTvl, overallStats.ActiveTvl)
	assert.Equal(t, expectedTotalTvl, overallStats.TotalTvl)
	assert.Equal(t, int64(2), overallStats.ActiveDelegations)
	assert.Equal(t, int64(3), overallStats.TotalDelegations)
}

func fetchFinalityEndpoint(t *testing.T, testServer *TestServer) []services.FpDetailsPublic {
	url := testServer.Server.URL + finalityProviderPath
	// Make a GET request to the finality providers endpoint
	resp, err := http.Get(url)
	assert.NoError(t, err, "making GET request to finality providers endpoint should not fail")
	defer resp.Body.Close()

	// Check that the status code is HTTP 200 OK
	assert.Equal(t, http.StatusOK, resp.StatusCode, "expected HTTP 200 OK status")

	// Read the response body
	bodyBytes, err := io.ReadAll(resp.Body)
	assert.NoError(t, err, "reading response body should not fail")

	var responseBody handlers.PublicResponse[[]services.FpDetailsPublic]
	err = json.Unmarshal(bodyBytes, &responseBody)
	assert.NoError(t, err, "unmarshalling response body should not fail")

	return responseBody.Data
}

func fetchOverallStatsEndpoint(t *testing.T, testServer *TestServer) services.StatsPublic {
	url := testServer.Server.URL + overallStatsEndpoint
	// Make a GET request to the stats endpoint
	resp, err := http.Get(url)
	assert.NoError(t, err, "making GET request to stats endpoint should not fail")
	defer resp.Body.Close()

	// Check that the status code is HTTP 200 OK
	assert.Equal(t, http.StatusOK, resp.StatusCode, "expected HTTP 200 OK status")

	// Read the response body
	bodyBytes, err := io.ReadAll(resp.Body)
	assert.NoError(t, err, "reading response body should not fail")

	var responseBody handlers.PublicResponse[services.StatsPublic]
	err = json.Unmarshal(bodyBytes, &responseBody)
	assert.NoError(t, err, "unmarshalling response body should not fail")

	return responseBody.Data
}
