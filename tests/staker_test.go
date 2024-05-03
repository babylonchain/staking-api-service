package tests

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"
	"fmt"

	"github.com/babylonchain/staking-api-service/internal/api/handlers"
	"github.com/babylonchain/staking-api-service/internal/services"
	"github.com/babylonchain/staking-api-service/internal/types"
	"github.com/stretchr/testify/assert"
)

func TestActiveStakingFetchedByStakerPkWithPaginationResponse(t *testing.T) {
	activeStakingEvent := buildActiveStakingEvent(mockStakerHash, 11)
	testServer := setupTestServer(t, nil)
	defer testServer.Close()
	sendTestMessage(testServer.Queues.ActiveStakingQueueClient, activeStakingEvent)
	// Wait for 2 seconds to make sure the message is processed
	time.Sleep(2 * time.Second)

	// Test the API
	url := testServer.Server.URL + stakerDelegations + "?staker_btc_pk=" + activeStakingEvent[0].StakerPkHex
	var paginationKey string
	allDataCollected := make([]services.DelegationPublic, 0)

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

		allDataCollected = append(allDataCollected, response.Data...)

		// Check if there's a next page
		if response.Pagination.NextKey == "" {
			t.Log("Already last page")
			break
		} else {
			t.Logf("Next page: %v", response.Pagination.NextKey)
			paginationKey = response.Pagination.NextKey
		}
	}

	assert.Greater(t, len(allDataCollected), 10, "expected more than 10 items in total across all pages")
	assert.NotEmpty(t, allDataCollected, "expected collected data to not be empty")

	for i := 0; i < len(allDataCollected)-1; i++ {
		assert.True(t, allDataCollected[i].StakingTx.StartHeight >= allDataCollected[i+1].StakingTx.StartHeight, "expected collected data to be sorted by start height")
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

	var response types.Error
	err = json.Unmarshal(bodyBytes, &response)
	assert.NoError(t, err, "unmarshalling response body should not fail")
}