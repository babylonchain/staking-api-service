package tests

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/babylonchain/staking-api-service/internal/api/handlers"
	"github.com/babylonchain/staking-api-service/internal/queue"
	"github.com/babylonchain/staking-api-service/internal/services"
	testmock "github.com/babylonchain/staking-api-service/tests/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	finalityProviderPath = "/v1/finality-providers"
)

func shouldGetFinalityProvidersSuccessfully(t *testing.T, server *httptest.Server, queue *queue.Queues) {
	url := server.URL + finalityProviderPath
	defer server.Close()
	defer queue.StopReceivingMessages()
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

	result := responseBody.Data
	// Check that the response body is as expected

	assert.NotEmpty(t, result, "expected response body to be non-empty")
	assert.Equal(t, "Babylon Foundation 2", result[2].Description.Moniker)
	assert.Equal(t, "0.060000000000000000", result[1].Commission)
	assert.Equal(t, "0d2f9728abc45c0cdeefdd73f52a0e0102470e35fb689fc5bc681959a61b021f", result[3].BtcPk)

	assert.Equal(t, 4, len(result))

	// Default to 0 as we have not yet implemented the logic to calculate these values
	assert.Equal(t, uint64(0), result[0].ActiveTvl)
	assert.Equal(t, uint64(0), result[0].TotalTvl)
	assert.Equal(t, uint64(0), result[0].ActiveDelegations)
	assert.Equal(t, uint64(0), result[0].TotalDelegations)
}

func TestGetFinalityProvidersSuccessfully(t *testing.T) {
	server, queue := setupTestServer(t, nil)
	shouldGetFinalityProvidersSuccessfully(t, server, queue)
}

func TestGetFinalityProviderShouldNotFailInCaseOfDbFailure(t *testing.T) {
	mockDB := new(testmock.DBClient)
	mockDB.On("FindFinalityProvidersByPkHex", mock.Anything, mock.Anything).Return(nil, errors.New("just an error"))

	server, queue := setupTestServer(t, &TestServerDependency{MockDbClient: mockDB})
	shouldGetFinalityProvidersSuccessfully(t, server, queue)
}
