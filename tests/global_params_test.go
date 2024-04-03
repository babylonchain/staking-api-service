package tests

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/babylonchain/staking-api-service/internal/api/handlers"
	"github.com/babylonchain/staking-api-service/internal/services"
	"github.com/stretchr/testify/assert"
)

const (
	globalParamsPath = "/v1/global-params"
)

func TestGlobalParams(t *testing.T) {
	server, queue := setupTestServer(t, nil)
	defer server.Close()
	defer queue.StopReceivingMessages()

	url := server.URL + globalParamsPath

	// Make a GET request to the health check endpoint
	resp, err := http.Get(url)
	assert.NoError(t, err, "making GET request to global params endpoint should not fail")
	defer resp.Body.Close()

	// Check that the status code is HTTP 200 OK
	assert.Equal(t, http.StatusOK, resp.StatusCode, "expected HTTP 200 OK status")

	// Read the response body
	bodyBytes, err := io.ReadAll(resp.Body)
	assert.NoError(t, err, "reading response body should not fail")

	var responseBody handlers.PublicResponse[services.GlobalParamsPublic]
	err = json.Unmarshal(bodyBytes, &responseBody)
	assert.NoError(t, err, "unmarshalling response body should not fail")

	result := responseBody.Data
	// Check that the response body is as expected
	assert.NotEmpty(t, result, "expected response body to be non-empty")
	assert.Equal(t, "bbt4", result.Tag)
	assert.Equal(t, 5, len(result.CovenantPks))
	assert.Equal(t, 4, len(result.FinalityProviders))
	assert.Equal(t, "Babylon Foundation 2", result.FinalityProviders[2].Description.Moniker)
	assert.Equal(t, "0.060000000000000000", result.FinalityProviders[1].Commission)
	assert.Equal(t, "0d2f9728abc45c0cdeefdd73f52a0e0102470e35fb689fc5bc681959a61b021f", result.FinalityProviders[3].BtcPk)

	assert.Equal(t, uint64(3), result.CovenantQuorum)
	assert.Equal(t, uint64(1000), result.UnbondingTime)
	assert.Equal(t, uint64(300000), result.MaxStakingAmount)
	assert.Equal(t, uint64(3000), result.MinStakingAmount)
	assert.Equal(t, uint64(10000), result.MaxStakingTime)
	assert.Equal(t, uint64(100), result.MinStakingTime)
}
