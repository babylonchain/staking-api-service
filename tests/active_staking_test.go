package tests

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/babylonchain/staking-api-service/internal/api/handlers"
	"github.com/babylonchain/staking-api-service/internal/services"
	"github.com/stretchr/testify/assert"
)

const (
	stakerDelegations = "/v1/staker/delegations"
	mockStakerHash    = "0x1234567890abcdef"
)

func TestActiveStaking(t *testing.T) {
	activeStakingEvent := buildActiveStakingEvent(mockStakerHash, 1)
	server, queues := setupTestServer(t, nil)
	defer server.Close()
	sendTestMessage(queues.ActiveStakingQueueClient, activeStakingEvent)

	// Wait for 2 seconds to make sure the message is processed
	time.Sleep(2 * time.Second)
	// Test the API
	url := server.URL + stakerDelegations + "?staker_btc_pk=" + activeStakingEvent[0].StakerPkHex
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
	assert.Equal(t, activeStakingEvent[0].StakingStartTimestamp, response.Data[0].StakingTx.StartTimestamp, "expected response body to match")
	assert.Equal(t, activeStakingEvent[0].StakingTimeLock, response.Data[0].StakingTx.TimeLock, "expected response body to match")
	assert.Equal(t, "active", response.Data[0].State, "expected response body to match")
	assert.Nil(t, response.Data[0].UnbondingTx, "expected response body to match")
	assert.Equal(t, activeStakingEvent[0].StakingTxHashHex, response.Data[0].StakingTxHashHex, "expected response body to match")

	assert.Empty(t, response.Pagination.NextKey, "should not have pagination")
}

func TestActiveStakingFetchedByStakerPkWithPaginationResponse(t *testing.T) {
	activeStakingEvent := buildActiveStakingEvent(mockStakerHash, 11)
	server, queues := setupTestServer(t, nil)
	defer server.Close()
	sendTestMessage(queues.ActiveStakingQueueClient, activeStakingEvent)

	// Wait for 2 seconds to make sure the message is processed
	time.Sleep(2 * time.Second)
	// Test the API
	url := server.URL + stakerDelegations + "?staker_btc_pk=" + activeStakingEvent[0].StakerPkHex
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
	assert.Equal(t, activeStakingEvent[0].StakerPkHex, response.Data[0].StakerPkHex, "expected response body to match")
	assert.Equal(t, 10, len(response.Data), "expected contain 10 items in response")
	assert.NotEmpty(t, response.Pagination.NextKey, "should have pagination token")
}
