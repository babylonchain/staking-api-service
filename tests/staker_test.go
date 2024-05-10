package tests

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/babylonchain/staking-api-service/internal/api"
	"github.com/babylonchain/staking-api-service/internal/api/handlers"
	"github.com/babylonchain/staking-api-service/internal/services"
	"github.com/stretchr/testify/assert"
)

func TestActiveStakingFetchedByStakerPkWithPaginationResponse(t *testing.T) {
	activeStakingEvent := buildActiveStakingEvent(mockStakerHash, 11)
	// randomly set one of the staking tx to be overflow
	activeStakingEvent[7].IsOverflow = true

	testServer := setupTestServer(t, nil)
	defer testServer.Close()
	sendTestMessage(testServer.Queues.ActiveStakingQueueClient, activeStakingEvent)
	// Wait for 2 seconds to make sure the message is processed
	time.Sleep(2 * time.Second)

	// Test the API
	url := testServer.Server.URL + stakerDelegations + "?staker_btc_pk=" + activeStakingEvent[0].StakerPkHex
	var paginationKey string
	allDataCollected := make([]services.DelegationPublic, 0)
	isFirstLoop := true

	// loop through all pages
	for {
		resp, err := http.Get(url + "&pagination_key=" + paginationKey)
		assert.NoError(t, err, "making GET request to delegations by staker pk should not fail")
		defer resp.Body.Close()

		// Check that the status code is 200 OK
		assert.Equal(t, http.StatusOK, resp.StatusCode, "expected HTTP 200 OK status")

		bodyBytes, err := io.ReadAll(resp.Body)
		assert.NoError(t, err, "reading response body should not fail")
		var response handlers.PublicResponse[[]services.DelegationPublic]
		err = json.Unmarshal(bodyBytes, &response)
		assert.NoError(t, err, "unmarshalling response body should not fail")

		// Check that the response body is as expected
		assert.NotEmptyf(t, response.Data, "expected response body to have data")
		assert.Equal(t, activeStakingEvent[0].StakerPkHex, response.Data[0].StakerPkHex, "expected response body to match")

		// check the timestamp string is in ISO format
		_, err = time.Parse(time.RFC3339, response.Data[0].StakingTx.StartTimestamp)
		assert.NoError(t, err, "expected timestamp to be in RFC3339 format")

		allDataCollected = append(allDataCollected, response.Data...)

		if isFirstLoop {
			assert.NotEmptyf(t, response.Pagination.NextKey, "should have pagination token after first iteration")
			isFirstLoop = false
		}

		if response.Pagination.NextKey != "" {
			t.Logf("Next page: %v", response.Pagination.NextKey)
			paginationKey = response.Pagination.NextKey
		} else {
			t.Log("Already last page")
			break
		}
	}

	assert.Greater(t, len(allDataCollected), 10, "expected more than 10 items in total across all pages")
	assert.NotEmptyf(t, allDataCollected, "expected collected data to not be empty")

	for i := 0; i < len(allDataCollected)-1; i++ {
		assert.True(t, allDataCollected[i].StakingTx.StartHeight >= allDataCollected[i+1].StakingTx.StartHeight, "expected collected data to be sorted by start height")
	}

	for _, d := range allDataCollected {
		if d.StakingTxHashHex == activeStakingEvent[7].StakingTxHashHex {
			assert.Equal(t, true, d.IsOverflow)
		} else {
			assert.Equal(t, false, d.IsOverflow)
		}
	}
}

func TestActiveStakingFetchedByStakerPkWithInvalidPaginationKey(t *testing.T) {
	activeStakingEvent := buildActiveStakingEvent(mockStakerHash, 11)
	testServer := setupTestServer(t, nil)
	defer testServer.Close()
	sendTestMessage(testServer.Queues.ActiveStakingQueueClient, activeStakingEvent)
	// Wait for 2 seconds to make sure the message is processed
	time.Sleep(2 * time.Second)

	// Test the API with an invalid pagination key
	url := fmt.Sprintf("%s%s?staker_btc_pk=%s&pagination_key=%s", testServer.Server.URL, stakerDelegations, activeStakingEvent[0].StakerPkHex, "btc_to_one_milly")

	resp, err := http.Get(url)
	assert.NoError(t, err, "making GET request to delegations by staker pk should not fail")
	defer resp.Body.Close()

	// Check that the status code is 400 Bad Request
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "expected HTTP 400 Bad Request status")

	bodyBytes, err := io.ReadAll(resp.Body)
	assert.NoError(t, err, "reading response body should not fail")

	var response api.ErrorResponse
	err = json.Unmarshal(bodyBytes, &response)
	assert.NoError(t, err, "unmarshalling response body should not fail")

	assert.Equal(t, "Invalid pagination token", response.Message, "expected error message does not match")
}
