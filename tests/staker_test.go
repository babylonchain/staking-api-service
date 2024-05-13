package tests

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"testing"
	"time"

	"github.com/babylonchain/staking-api-service/internal/api"
	"github.com/babylonchain/staking-api-service/internal/api/handlers"
	"github.com/babylonchain/staking-api-service/internal/services"
	"github.com/stretchr/testify/assert"
)

func FuzzTestStakerDelegationsWithPaginationResponse(f *testing.F) {
	attachRandomSeedsToFuzzer(f, 3)
	f.Fuzz(func(t *testing.T, seed int64) {
		r := rand.New(rand.NewSource(seed))
		opts := &TestActiveEventGeneratorOpts{
			NumOfEvents:     11,
			NumberOfFps:     randomPositiveInt(r, 11),
			NumberOfStakers: 1,
		}
		activeStakingEventsByStaker1 := generateRandomActiveStakingEvents(t, r, opts)
		activeStakingEventsByStaker2 := generateRandomActiveStakingEvents(t, r, opts)
		testServer := setupTestServer(t, nil)
		defer testServer.Close()
		sendTestMessage(
			testServer.Queues.ActiveStakingQueueClient,
			append(activeStakingEventsByStaker1, activeStakingEventsByStaker2...),
		)
		time.Sleep(5 * time.Second)

		// Test the API
		stakerPk := activeStakingEventsByStaker1[0].StakerPkHex
		url := testServer.Server.URL + stakerDelegations + "?staker_btc_pk=" + stakerPk
		var paginationKey string
		var allDataCollected []services.DelegationPublic
		var atLeastOnePage bool
		for {
			resp, err := http.Get(url + "&pagination_key=" + paginationKey)
			assert.NoError(t, err, "making GET request to delegations by staker pk should not fail")
			assert.Equal(t, http.StatusOK, resp.StatusCode, "expected HTTP 200 OK status")
			bodyBytes, err := io.ReadAll(resp.Body)
			assert.NoError(t, err, "reading response body should not fail")
			var response handlers.PublicResponse[[]services.DelegationPublic]
			err = json.Unmarshal(bodyBytes, &response)
			assert.NoError(t, err, "unmarshalling response body should not fail")

			// Check that the response body is as expected
			assert.NotEmptyf(t, response.Data, "expected response body to have data")
			for _, d := range response.Data {
				assert.Equal(t, stakerPk, d.StakerPkHex, "expected response body to match")
			}
			allDataCollected = append(allDataCollected, response.Data...)
			if response.Pagination.NextKey != "" {
				paginationKey = response.Pagination.NextKey
				atLeastOnePage = true
			} else {
				break
			}
		}

		assert.True(t, atLeastOnePage, "expected at least one page of data")
		assert.Equal(t, 11, len(allDataCollected), "expected 11 items in total")
		for _, events := range activeStakingEventsByStaker1 {
			found := false
			for _, d := range allDataCollected {
				if d.StakingTxHashHex == events.StakingTxHashHex {
					found = true
					break
				}
			}
			assert.True(t, found, "expected to find the staking tx in the response")
		}
		for i := 0; i < len(allDataCollected)-1; i++ {
			assert.True(t, allDataCollected[i].StakingTx.StartHeight >= allDataCollected[i+1].StakingTx.StartHeight, "expected collected data to be sorted by start height")
		}
	})
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
