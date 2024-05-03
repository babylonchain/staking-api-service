package tests

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/babylonchain/staking-api-service/internal/api/handlers"
	"github.com/babylonchain/staking-api-service/internal/services"
	"github.com/babylonchain/staking-api-service/internal/db/model"
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
	initialPagination := model.DelegationByStakerPagination{
		StakingTxHashHex:   "4b688df6b7b39835f4565738939d6260f4c6f1624a4c1f26395af32bb464b451", // random hash placeholder
		StakingStartHeight: 100,
	}
	paginationToken, err := model.GetPaginationToken(initialPagination)
	assert.NoError(t, err, "generating pagination token should not fail")

	t.Logf("initial pagination token: %v", paginationToken)

	url := testServer.Server.URL + stakerDelegations + "?staker_btc_pk=" + activeStakingEvent[0].StakerPkHex + "&pagination_key=" + paginationToken
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
	assert.NotEmpty(t, response.Data, "expected response body to have data")
	assert.Equal(t, 10, len(response.Data), "expected contain 10 items in response")
	assert.Equal(t, activeStakingEvent[0].StakerPkHex, response.Data[0].StakerPkHex, "expected response body to match")
	// check the timestamp string is in ISO format
	_, err = time.Parse(time.RFC3339, response.Data[0].StakingTx.StartTimestamp)
	assert.NoError(t, err, "expected timestamp to be in RFC3339 format")

	t.Log("Checking pagination token and data...")
    if response.Pagination.NextKey == "" {
        t.Log("Warning: NextKey is empty. Pagination may not be available or is finished.")
    } else {
        decodedToken, err := model.DecodePaginationToken[model.DelegationByStakerPagination](response.Pagination.NextKey)
        assert.NoError(t, err, "decoding next page pagination token should not fail")
        if err != nil {
            t.Logf("Error decoding pagination token: %v", err)
            return
        }

        assert.True(t, decodedToken.StakingStartHeight > 100, "expected the next page start height to be greater than 100")
        if decodedToken.StakingStartHeight <= 100 {
            t.Logf("Unexpected start height: %d", decodedToken.StakingStartHeight)
        }
    }

		// Also make sure the returned data is sorted by staking start height
		for i := 0; i < len(response.Data)-1; i++ {
			assert.True(t, response.Data[i].StakingTx.StartHeight >= response.Data[i+1].StakingTx.StartHeight, "expected response body to be sorted")
		}
}
