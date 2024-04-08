package tests

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/babylonchain/staking-queue-client/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/babylonchain/staking-api-service/internal/api/handlers"
	"github.com/babylonchain/staking-api-service/internal/services"
)

const (
	stakerDelegations = "/v1/staker/delegations"
	mockStakerHash    = "0x1234567890abcdef"
)

func TestActiveStaking(t *testing.T) {
	activeStakingEvent := buildActiveStakingEvent(mockStakerHash, 1)
	testServer := setupTestServer(t, nil)
	defer testServer.Close()
	sendTestMessage(testServer.Queues.ActiveStakingQueueClient, activeStakingEvent)

	// Wait for 2 seconds to make sure the message is processed
	time.Sleep(2 * time.Second)
	// Test the API
	url := testServer.Server.URL + stakerDelegations + "?staker_btc_pk=" + activeStakingEvent[0].StakerPkHex
	resp, err := http.Get(url)
	assert.NoError(t, err, "making GET request to delegations by staker pk should not fail")
	defer resp.Body.Close()

	// Check that the status code is HTTP 200 OK
	assert.Equal(t, http.StatusOK, resp.StatusCode, "expected HTTP 200 OK status")

	// Read the response body
	bodyBytes, err := io.ReadAll(resp.Body)
	assert.NoError(t, err, "reading response body should not fail")

	var response handlers.PublicResponse[[]services.DelegationPublic]
	err = json.Unmarshal(bodyBytes, &response)
	assert.NoError(t, err, "unmarshalling response body should not fail")

	// Check that the response body is as expected
	assert.Equal(t, 1, len(response.Data), "expected contain 1 item in response")
	assert.Equal(t, activeStakingEvent[0].StakerPkHex, response.Data[0].StakerPkHex, "expected response body to match")
	assert.Equal(t, activeStakingEvent[0].StakingValue, response.Data[0].StakingValue, "expected response body to match")
	assert.Equal(t, activeStakingEvent[0].StakingTxHex, response.Data[0].StakingTx.TxHex, "expected response body to match")
	assert.Equal(t, activeStakingEvent[0].StakingOutputIndex, response.Data[0].StakingTx.OutputIndex, "expected response body to match")
	assert.Equal(t, activeStakingEvent[0].StakingStartHeight, response.Data[0].StakingTx.StartHeight, "expected response body to match")
	assert.Equal(t, activeStakingEvent[0].StakingTimeLock, response.Data[0].StakingTx.TimeLock, "expected response body to match")
	assert.Equal(t, "active", response.Data[0].State, "expected response body to match")
	assert.Nil(t, response.Data[0].UnbondingTx, "expected response body to match")
	assert.Equal(t, activeStakingEvent[0].StakingTxHashHex, response.Data[0].StakingTxHashHex, "expected response body to match")

	assert.Empty(t, response.Pagination.NextKey, "should not have pagination")
}

func TestUnbondActiveStaking(t *testing.T) {
	activeStakingEvent := buildActiveStakingEvent(mockStakerHash, 1)
	expiredStakingEvent := client.NewExpiredStakingEvent(activeStakingEvent[0].StakingTxHashHex, client.ActiveTxType)
	testServer := setupTestServer(t, nil)
	defer testServer.Close()
	sendTestMessage(testServer.Queues.ActiveStakingQueueClient, activeStakingEvent)
	time.Sleep(2 * time.Second)
	sendTestMessage(testServer.Queues.ExpiredStakingQueueClient, []client.ExpiredStakingEvent{expiredStakingEvent})
	time.Sleep(2 * time.Second)

	// Test the API
	url := testServer.Server.URL + stakerDelegations + "?staker_btc_pk=" + activeStakingEvent[0].StakerPkHex
	resp, err := http.Get(url)
	assert.NoError(t, err, "making GET request to delegations by staker pk should not fail")
	defer resp.Body.Close()

	// Check that the status code is HTTP 200 OK
	assert.Equal(t, http.StatusOK, resp.StatusCode, "expected HTTP 200 OK status")

	// Read the response body
	bodyBytes, err := io.ReadAll(resp.Body)
	assert.NoError(t, err, "reading response body should not fail")

	var response handlers.PublicResponse[[]services.DelegationPublic]
	err = json.Unmarshal(bodyBytes, &response)
	assert.NoError(t, err, "unmarshalling response body should not fail")

	// Check that the response body is as expected
	assert.Equal(t, 1, len(response.Data), "expected contain 1 item in response")
	assert.Equal(t, "unbonded", response.Data[0].State)
}

func TestUnbondActiveStakingShouldTolerateOutOfOrder(t *testing.T) {
	activeStakingEvent := buildActiveStakingEvent(mockStakerHash, 1)
	expiredStakingEvent := client.NewExpiredStakingEvent(activeStakingEvent[0].StakingTxHashHex, client.ActiveTxType)
	testServer := setupTestServer(t, nil)
	defer testServer.Close()
	sendTestMessage(testServer.Queues.ExpiredStakingQueueClient, []client.ExpiredStakingEvent{expiredStakingEvent})
	time.Sleep(2 * time.Second)
	sendTestMessage(testServer.Queues.ActiveStakingQueueClient, activeStakingEvent)
	time.Sleep(10 * time.Second)

	// Test the API
	url := testServer.Server.URL + stakerDelegations + "?staker_btc_pk=" + activeStakingEvent[0].StakerPkHex
	resp, err := http.Get(url)
	assert.NoError(t, err, "making GET request to delegations by staker pk should not fail")
	defer resp.Body.Close()

	// Check that the status code is HTTP 200 OK
	assert.Equal(t, http.StatusOK, resp.StatusCode, "expected HTTP 200 OK status")

	// Read the response body
	bodyBytes, err := io.ReadAll(resp.Body)
	assert.NoError(t, err, "reading response body should not fail")

	var response handlers.PublicResponse[[]services.DelegationPublic]
	err = json.Unmarshal(bodyBytes, &response)
	assert.NoError(t, err, "unmarshalling response body should not fail")

	// Check that the response body is as expected
	assert.Equal(t, 1, len(response.Data), "expected contain 1 item in response")
	assert.Equal(t, "unbonded", response.Data[0].State)
}

func TestShouldNotUnbondIfNotActiveState(t *testing.T) {
	activeStakingEvent := getTestActiveStakingEvent()
	expiredStakingEvent := client.NewExpiredStakingEvent(activeStakingEvent.StakingTxHashHex, client.ActiveTxType)
	testServer := setupTestServer(t, nil)
	defer testServer.Close()
	err := sendTestMessage(testServer.Queues.ActiveStakingQueueClient, []client.ActiveStakingEvent{activeStakingEvent})
	require.NoError(t, err)
	time.Sleep(2 * time.Second)

	// Let's make a POST request to the unbonding endpoint to change the state to unbonding_requested
	unbondingUrl := testServer.Server.URL + unbondingPath
	requestBody := getTestUnbondDelegationRequestPayload(activeStakingEvent.StakingTxHashHex)
	requestBodyBytes, err := json.Marshal(requestBody)
	assert.NoError(t, err, "marshalling request body should not fail")

	resp, err := http.Post(unbondingUrl, "application/json", bytes.NewReader(requestBodyBytes))
	assert.NoError(t, err, "making POST request to unbonding endpoint should not fail")
	defer resp.Body.Close()

	// Check that the status code is HTTP 202
	assert.Equal(t, http.StatusAccepted, resp.StatusCode, "expected HTTP 202 Accepted status")

	sendTestMessage(testServer.Queues.ExpiredStakingQueueClient, []client.ExpiredStakingEvent{expiredStakingEvent})
	time.Sleep(2 * time.Second)

	// Test the API
	url := testServer.Server.URL + stakerDelegations + "?staker_btc_pk=" + activeStakingEvent.StakerPkHex
	resp, err = http.Get(url)
	assert.NoError(t, err, "making GET request to delegations by staker pk should not fail")
	defer resp.Body.Close()

	// Check that the status code is HTTP 200 OK
	assert.Equal(t, http.StatusOK, resp.StatusCode, "expected HTTP 200 OK status")

	// Read the response body
	bodyBytes, err := io.ReadAll(resp.Body)
	assert.NoError(t, err, "reading response body should not fail")

	var response handlers.PublicResponse[[]services.DelegationPublic]
	err = json.Unmarshal(bodyBytes, &response)
	assert.NoError(t, err, "unmarshalling response body should not fail")

	// Check that the response body is as expected
	assert.Equal(t, 1, len(response.Data), "expected contain 1 item in response")
	// The state should not be updated to unbonded
	assert.Equal(t, "unbonding_requested", response.Data[0].State)
}
