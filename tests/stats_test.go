package tests

import (
	"bytes"
	"encoding/json"
	"io"
	"math/rand"
	"net/http"
	"testing"
	"time"

	"github.com/babylonchain/staking-api-service/internal/api/handlers"
	"github.com/babylonchain/staking-api-service/internal/db/model"
	"github.com/babylonchain/staking-api-service/internal/services"
	"github.com/babylonchain/staking-queue-client/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	overallStatsEndpoint = "/v1/stats"
	topStakerStatsPath   = "/v1/stats/staker"
)

func TestStatsShouldBeShardedInDb(t *testing.T) {
	activeStakingEvent := buildActiveStakingEvent(mockStakerHash, 10)
	// build the unbonding event based on the active staking event
	var unbondingEvents []client.UnbondingStakingEvent
	for _, event := range activeStakingEvent {
		unbondingEvents = append(unbondingEvents, client.NewUnbondingStakingEvent(
			event.StakingTxHashHex,
			event.StakingStartHeight+100,
			time.Now().Unix(),
			10,
			1,
			event.StakingTxHex,     // mocked data, it doesn't matter in stats calculation
			event.StakingTxHashHex, // mocked data, it doesn't matter in stats calculation
		))
	}
	testServer := setupTestServer(t, nil)
	defer testServer.Close()
	sendTestMessage(testServer.Queues.ActiveStakingQueueClient, activeStakingEvent)
	time.Sleep(2 * time.Second)
	sendTestMessage(testServer.Queues.UnbondingStakingQueueClient, unbondingEvents)
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
}

func TestShouldSkipStatsCalculationForOverflowedStakingEvent(t *testing.T) {
	activeStakingEvent := getTestActiveStakingEvent()
	// Set the overflow flag to true
	activeStakingEvent.IsOverflow = true
	testServer := setupTestServer(t, nil)
	defer testServer.Close()

	err := sendTestMessage(testServer.Queues.ActiveStakingQueueClient, []client.ActiveStakingEvent{*activeStakingEvent})
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
	assert.Equal(t, 0, len(stats))
}

func TestShouldNotPerformStatsCalculationForUnbondingTxWhenDelegationIsOverflowed(t *testing.T) {
	activeStakingEvent := buildActiveStakingEvent(mockStakerHash, 10)
	// Let's pick a random staking event and set the overflow flag to true
	event := activeStakingEvent[6]
	event.IsOverflow = true
	// build the unbonding event based on the active staking event
	var unbondingEvents []client.UnbondingStakingEvent
	unbondingEvents = append(unbondingEvents, client.NewUnbondingStakingEvent(
		event.StakingTxHashHex,
		event.StakingStartHeight+100,
		time.Now().Unix(),
		10,
		1,
		event.StakingTxHex,     // mocked data, it doesn't matter in stats calculation
		event.StakingTxHashHex, // mocked data, it doesn't matter in stats calculation
	))
	testServer := setupTestServer(t, nil)
	defer testServer.Close()
	sendTestMessage(testServer.Queues.ActiveStakingQueueClient, activeStakingEvent)
	time.Sleep(2 * time.Second)
	sendTestMessage(testServer.Queues.UnbondingStakingQueueClient, unbondingEvents)
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

	// calculate the total expect tvl from the active staking events
	var expectedTotalTvl int64
	for _, e := range activeStakingEvent {
		if !e.IsOverflow {
			expectedTotalTvl += int64(e.StakingValue)
		}
	}
	assert.Equal(t, expectedTotalTvl, totalActiveTvl)
	assert.Equal(t, int64(9), totalActiveDelegations)
	assert.Equal(t, expectedTotalTvl, totalTvl)
	assert.Equal(t, int64(9), totalDelegations)
}

func TestStatsEndpoints(t *testing.T) {
	activeStakingEvent := getTestActiveStakingEvent()
	testServer := setupTestServer(t, nil)
	defer testServer.Close()
	sendTestMessage(testServer.Queues.ActiveStakingQueueClient, []client.ActiveStakingEvent{*activeStakingEvent})
	time.Sleep(2 * time.Second)

	// Test the finality endpoint first
	result := fetchFinalityEndpoint(t, testServer)
	assert.Equal(t, 4, len(result))
	for _, r := range result {
		if r.BtcPk == activeStakingEvent.FinalityProviderPkHex {
			assert.Equal(t, int64(activeStakingEvent.StakingValue), r.ActiveTvl)
			assert.Equal(t, int64(activeStakingEvent.StakingValue), r.TotalTvl)
			assert.Equal(t, int64(1), r.ActiveDelegations)
			assert.Equal(t, int64(1), r.TotalDelegations)
		} else {
			assert.Equal(t, int64(0), r.ActiveTvl)
			assert.Equal(t, int64(0), r.TotalTvl)
			assert.Equal(t, int64(0), r.ActiveDelegations)
			assert.Equal(t, int64(0), r.TotalDelegations)
		}
	}

	// Test the overall stats endpoint
	overallStats := fetchOverallStatsEndpoint(t, testServer)
	assert.Equal(t, int64(activeStakingEvent.StakingValue), overallStats.ActiveTvl)
	assert.Equal(t, int64(activeStakingEvent.StakingValue), overallStats.TotalTvl)
	assert.Equal(t, int64(1), overallStats.ActiveDelegations)
	assert.Equal(t, int64(1), overallStats.TotalDelegations)
	assert.Equal(t, uint64(1), overallStats.TotalStakers)
	// We have not yet sent any UnconfirmedInfoEvent, hence no recrod in db
	assert.Equal(t, uint64(0), overallStats.UnconfirmedActiveTvl)

	// Test the top staker stats endpoint
	stakerStats, _ := fetchStakerStatsEndpoint(t, testServer)
	assert.Equal(t, 1, len(stakerStats))
	assert.Equal(t, activeStakingEvent.StakerPkHex, stakerStats[0].StakerPkHex)
	assert.Equal(t, int64(activeStakingEvent.StakingValue), stakerStats[0].ActiveTvl)
	assert.Equal(t, int64(activeStakingEvent.StakingValue), stakerStats[0].TotalTvl)
	assert.Equal(t, int64(1), stakerStats[0].ActiveDelegations)
	assert.Equal(t, int64(1), stakerStats[0].TotalDelegations)

	// Now let's send an unbonding event
	unbondingEvent := client.NewUnbondingStakingEvent(
		activeStakingEvent.StakingTxHashHex,
		activeStakingEvent.StakingStartHeight+100,
		time.Now().Unix(),
		10,
		1,
		activeStakingEvent.StakingTxHex,     // mocked data, it doesn't matter in stats calculation
		activeStakingEvent.StakingTxHashHex, // mocked data, it doesn't matter in stats calculation
	)
	sendTestMessage(testServer.Queues.UnbondingStakingQueueClient, []client.UnbondingStakingEvent{unbondingEvent})
	time.Sleep(2 * time.Second)

	// Make a GET request to the finality providers endpoint
	result = fetchFinalityEndpoint(t, testServer)
	assert.Equal(t, 4, len(result))
	for _, r := range result {
		if r.BtcPk == activeStakingEvent.FinalityProviderPkHex {
			assert.Equal(t, int64(0), r.ActiveTvl)
			assert.Equal(t, int64(activeStakingEvent.StakingValue), r.TotalTvl)
			assert.Equal(t, int64(0), r.ActiveDelegations)
			assert.Equal(t, int64(1), r.TotalDelegations)
		} else {
			assert.Equal(t, int64(0), r.ActiveTvl)
			assert.Equal(t, int64(0), r.TotalTvl)
			assert.Equal(t, int64(0), r.ActiveDelegations)
			assert.Equal(t, int64(0), r.TotalDelegations)
		}
	}

	overallStats = fetchOverallStatsEndpoint(t, testServer)
	assert.Equal(t, int64(0), overallStats.ActiveTvl)
	assert.Equal(t, int64(activeStakingEvent.StakingValue), overallStats.TotalTvl)
	assert.Equal(t, int64(0), overallStats.ActiveDelegations)
	assert.Equal(t, int64(1), overallStats.TotalDelegations)
	assert.Equal(t, uint64(1), overallStats.TotalStakers)

	stakerStats, _ = fetchStakerStatsEndpoint(t, testServer)
	assert.Equal(t, 1, len(stakerStats))
	assert.Equal(t, activeStakingEvent.StakerPkHex, stakerStats[0].StakerPkHex)
	assert.Equal(t, int64(0), stakerStats[0].ActiveTvl)
	assert.Equal(t, int64(activeStakingEvent.StakingValue), stakerStats[0].TotalTvl)
	assert.Equal(t, int64(0), stakerStats[0].ActiveDelegations)
	assert.Equal(t, int64(1), stakerStats[0].TotalDelegations)

	// Send two new active events, it will increment the stats
	activeEvents := buildActiveStakingEvent(mockStakerHash, 2)
	sendTestMessage(testServer.Queues.ActiveStakingQueueClient, activeEvents)
	time.Sleep(2 * time.Second)

	// Make a GET request to the finality providers endpoint
	finalityProviderStats := fetchFinalityEndpoint(t, testServer)
	assert.Equal(t, 6, len(finalityProviderStats))
	// Make sure sorted by active TVL
	for i := 0; i < len(finalityProviderStats)-1; i++ {
		assert.True(t, finalityProviderStats[i].ActiveTvl >= finalityProviderStats[i+1].ActiveTvl, "expected response body to be sorted")
	}

	overallStats = fetchOverallStatsEndpoint(t, testServer)

	expectedTvl := int64(activeEvents[0].StakingValue + activeEvents[1].StakingValue)
	expectedTotalTvl := int64(expectedTvl) + int64(activeStakingEvent.StakingValue)
	assert.Equal(t, expectedTvl, overallStats.ActiveTvl)
	assert.Equal(t, expectedTotalTvl, overallStats.TotalTvl)
	assert.Equal(t, int64(2), overallStats.ActiveDelegations)
	assert.Equal(t, int64(3), overallStats.TotalDelegations)
	assert.Equal(t, uint64(2), overallStats.TotalStakers, "expected 2 stakers as the last 2 belong to same staker")

	stakerStats, _ = fetchStakerStatsEndpoint(t, testServer)
	assert.Equal(t, 2, len(stakerStats))

	// Also make sure the returned data is sorted by active TVL
	for i := 0; i < len(stakerStats)-1; i++ {
		assert.True(t, stakerStats[i].ActiveTvl >= stakerStats[i+1].ActiveTvl, "expected response body to be sorted")
	}

	// send an UnconfirmedInfoEvent which shall update the unconfirmed active TVL
	unconfirmedInfoEvent := &client.UnconfirmedInfoEvent{
		EventType: client.UnconfirmedInfoEventType,
		Height:    100,
		ActiveTvl: 100,
	}
	sendTestMessage(testServer.Queues.UnconfirmedInfoQueueClient, []*client.UnconfirmedInfoEvent{unconfirmedInfoEvent})

	time.Sleep(2 * time.Second)

	overallStats = fetchOverallStatsEndpoint(t, testServer)
	assert.Equal(t, uint64(100), overallStats.UnconfirmedActiveTvl)
}

func FuzzStatsEndpointReturnHighestUnconfirmedTvlFromEvents(f *testing.F) {
	attachRandomSeedsToFuzzer(f, 5)
	f.Fuzz(func(t *testing.T, seed int64) {
		r := rand.New(rand.NewSource(seed))
		testServer := setupTestServer(t, nil)
		defer testServer.Close()

		overallStats := fetchOverallStatsEndpoint(t, testServer)
		assert.Equal(t, uint64(0), overallStats.UnconfirmedActiveTvl)

		highestHeightEvent := &client.UnconfirmedInfoEvent{
			EventType: client.UnconfirmedInfoEventType,
			Height:    0,
			ActiveTvl: 0,
		}
		var messages []*client.UnconfirmedInfoEvent
		for i := 0; i < 10; i++ {
			unconfirmedInfoEvent := &client.UnconfirmedInfoEvent{
				EventType: client.UnconfirmedInfoEventType,
				Height:    randomBtcHeight(r, 0),
				ActiveTvl: uint64(randomAmount(r)),
			}
			messages = append(messages, unconfirmedInfoEvent)
			if unconfirmedInfoEvent.Height > highestHeightEvent.Height {
				highestHeightEvent = unconfirmedInfoEvent
			}
		}
		sendTestMessage(testServer.Queues.UnconfirmedInfoQueueClient, messages)
		time.Sleep(5 * time.Second)

		overallStats = fetchOverallStatsEndpoint(t, testServer)
		assert.Equal(t, &highestHeightEvent.ActiveTvl, &overallStats.UnconfirmedActiveTvl)
	})

}

func fetchFinalityEndpoint(t *testing.T, testServer *TestServer) []services.FpDetailsPublic {
	url := testServer.Server.URL + finalityProvidersPath
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

func fetchOverallStatsEndpoint(t *testing.T, testServer *TestServer) services.OverallStatsPublic {
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

	var responseBody handlers.PublicResponse[services.OverallStatsPublic]
	err = json.Unmarshal(bodyBytes, &responseBody)
	assert.NoError(t, err, "unmarshalling response body should not fail")

	return responseBody.Data
}

func fetchStakerStatsEndpoint(t *testing.T, testServer *TestServer) ([]services.StakerStatsPublic, string) {
	url := testServer.Server.URL + topStakerStatsPath
	resp, err := http.Get(url)
	assert.NoError(t, err, "making GET request to staker stats endpoint should not fail")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "expected HTTP 200 OK status")

	// Read the response body
	bodyBytes, err := io.ReadAll(resp.Body)
	assert.NoError(t, err, "reading response body should not fail")

	var responseBody handlers.PublicResponse[[]services.StakerStatsPublic]
	err = json.Unmarshal(bodyBytes, &responseBody)
	assert.NoError(t, err, "unmarshalling response body should not fail")

	return responseBody.Data, responseBody.Pagination.NextKey
}
