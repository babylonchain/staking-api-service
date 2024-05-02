package tests

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/babylonchain/staking-api-service/internal/api/handlers"
	"github.com/babylonchain/staking-api-service/internal/services"
)

const (
	globalParamsPath = "/v1/global-params"
)

func TestGlobalParams(t *testing.T) {
	testServer := setupTestServer(t, nil)
	defer testServer.Close()

	url := testServer.Server.URL + globalParamsPath

	// Make a GET request to the global params endpoint
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

	result := responseBody.Data.Versions
	// Check that the response body is as expected
	assert.NotEmpty(t, result, "expected response body to be non-empty")
	assert.Equal(t, 1, len(result))
	versionedGlobalParam := result[0]

	assert.Equal(t, uint64(0), versionedGlobalParam.Version)
	assert.Equal(t, uint64(100), versionedGlobalParam.ActivationHeight)
	assert.Equal(t, uint64(50), versionedGlobalParam.StakingCap)
	assert.Equal(t, "bbt4", versionedGlobalParam.Tag)
	assert.Equal(t, 5, len(versionedGlobalParam.CovenantPks))
	assert.Equal(t, uint64(3), versionedGlobalParam.CovenantQuorum)
	assert.Equal(t, uint64(1000), versionedGlobalParam.UnbondingTime)
	assert.Equal(t, uint64(10000), versionedGlobalParam.UnbondingFee)
	assert.Equal(t, uint64(300000), versionedGlobalParam.MaxStakingAmount)
	assert.Equal(t, uint64(3000), versionedGlobalParam.MinStakingAmount)
	assert.Equal(t, uint64(10000), versionedGlobalParam.MaxStakingTime)
	assert.Equal(t, uint64(100), versionedGlobalParam.MinStakingTime)
}
