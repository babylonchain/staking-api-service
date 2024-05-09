package tests

import (
	"encoding/json"
	"errors"
	"io"
	"math/rand"
	"net/http"
	"testing"
	"time"

	"github.com/babylonchain/staking-api-service/internal/api/handlers"
	"github.com/babylonchain/staking-api-service/internal/db"
	"github.com/babylonchain/staking-api-service/internal/db/model"
	"github.com/babylonchain/staking-api-service/internal/services"
	"github.com/babylonchain/staking-api-service/internal/types"
	testmock "github.com/babylonchain/staking-api-service/tests/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	finalityProvidersPath = "/v1/finality-providers"
)

func shouldGetFinalityProvidersSuccessfully(t *testing.T, testServer *TestServer) {
	url := testServer.Server.URL + finalityProvidersPath
	defer testServer.Close()
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

	assert.Equal(t, int64(0), result[0].ActiveTvl)
	assert.Equal(t, int64(0), result[0].TotalTvl)
	assert.Equal(t, int64(0), result[0].ActiveDelegations)
	assert.Equal(t, int64(0), result[0].TotalDelegations)
}

func TestGetFinalityProvidersSuccessfully(t *testing.T) {
	testServer := setupTestServer(t, nil)
	shouldGetFinalityProvidersSuccessfully(t, testServer)
}

func TestGetFinalityProviderShouldNotFailInCaseOfDbFailure(t *testing.T) {
	mockDB := new(testmock.DBClient)
	mockDB.On("FindFinalityProviderStats", mock.Anything, mock.Anything).Return(nil, errors.New("just an error"))

	testServer := setupTestServer(t, &TestServerDependency{MockDbClient: mockDB})
	shouldGetFinalityProvidersSuccessfully(t, testServer)
}

func TestGetFinalityProviderShouldReturnFallbackToGlobalParams(t *testing.T) {
	mockedResultMap := &db.DbResultMap[*model.FinalityProviderStatsDocument]{
		Data:            []*model.FinalityProviderStatsDocument{},
		PaginationToken: "",
	}
	mockDB := new(testmock.DBClient)
	mockDB.On("FindFinalityProviderStats", mock.Anything, mock.Anything).Return(mockedResultMap, nil)

	testServer := setupTestServer(t, &TestServerDependency{MockDbClient: mockDB})
	shouldGetFinalityProvidersSuccessfully(t, testServer)
}

func TestGetFinalityProviderReturn4xxErrorIfPageTokenInvalid(t *testing.T) {
	mockDB := new(testmock.DBClient)
	mockDB.On("FindFinalityProviderStats", mock.Anything, mock.Anything).Return(nil, &db.InvalidPaginationTokenError{})

	testServer := setupTestServer(t, &TestServerDependency{MockDbClient: mockDB})
	url := testServer.Server.URL + finalityProvidersPath
	defer testServer.Close()
	// Make a GET request to the finality providers endpoint
	resp, err := http.Get(url)
	assert.NoError(t, err, "making GET request to finality providers endpoint should not fail")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestGetFinalityProviderShouldReturnAllRegisteredFps(t *testing.T) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	fpParams := generateRandomFinalityProviderDetail(t, r, 10)
	mockDB := new(testmock.DBClient)
	fpRegistered := fpParams[r.Intn(10)].BtcPk
	fpNotRegistered, err := randomPk()
	assert.NoError(t, err, "generating random public key should not fail")

	fpRegisteredStats := &model.FinalityProviderStatsDocument{
		FinalityProviderPkHex: fpRegistered,
		ActiveTvl:             randomAmount(r),
		TotalTvl:              randomAmount(r),
		ActiveDelegations:     r.Int63n(100),
		TotalDelegations:      r.Int63n(1000),
	}
	fpNotRegisteredStats := &model.FinalityProviderStatsDocument{
		FinalityProviderPkHex: fpNotRegistered,
		ActiveTvl:             randomAmount(r),
		TotalTvl:              randomAmount(r),
		ActiveDelegations:     r.Int63n(100),
		TotalDelegations:      r.Int63n(1000),
	}

	mockedFinalityProviderStats := &db.DbResultMap[*model.FinalityProviderStatsDocument]{
		Data:            []*model.FinalityProviderStatsDocument{fpRegisteredStats, fpNotRegisteredStats},
		PaginationToken: "",
	}
	mockDB.On("FindFinalityProviderStats", mock.Anything, mock.Anything).Return(mockedFinalityProviderStats, nil)
	mockDB.On("FindFinalityProviderStatsByFinalityProviderPkHex",
		mock.Anything, mock.Anything,
	).Return([]*model.FinalityProviderStatsDocument{fpRegisteredStats}, nil)

	testServer := setupTestServer(t, &TestServerDependency{MockDbClient: mockDB, MockedFinalityProviders: fpParams})

	url := testServer.Server.URL + finalityProvidersPath
	defer testServer.Close()
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
	// We expect all registered finality providers to be returned, plus the one that is not registered
	assert.Equal(t, len(fpParams)+1, len(result))

	resultMap := make(map[string]services.FpDetailsPublic)
	for _, fp := range result {
		resultMap[fp.BtcPk] = fp
	}

	for _, f := range fpParams {
		assert.NotEmpty(t, resultMap[f.BtcPk])
		assert.Equal(t, f.Description.Moniker, resultMap[f.BtcPk].Description.Moniker)
		assert.Equal(t, f.Commission, resultMap[f.BtcPk].Commission)
		// Check that the stats are correct for the registered finality providers without any delegations
		if f.BtcPk != fpRegistered {
			assert.Equal(t, int64(0), resultMap[f.BtcPk].ActiveTvl)
			assert.Equal(t, int64(0), resultMap[f.BtcPk].TotalTvl)
			assert.Equal(t, int64(0), resultMap[f.BtcPk].ActiveDelegations)
			assert.Equal(t, int64(0), resultMap[f.BtcPk].TotalDelegations)
		}
	}

	// Check that the fpRegistered fp also has the correct stats
	assert.Equal(t, int64(mockedFinalityProviderStats.Data[0].ActiveTvl), resultMap[fpRegistered].ActiveTvl)
	// The fpNotRegistered fp should not have any FP defails
	assert.Equal(t, "", resultMap[fpNotRegistered].Description.Moniker)
}

func TestGetFinalityProviderShouldNotReturnRegisteredFpWithoutStakingForPaginatedDbResponse(t *testing.T) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	fpParams := generateRandomFinalityProviderDetail(t, r, 10)
	mockDB := new(testmock.DBClient)
	fpRegistered := fpParams[r.Intn(10)].BtcPk
	fpNotRegistered, err := randomPk()
	assert.NoError(t, err, "generating random public key should not fail")

	fpRegisteredStats := &model.FinalityProviderStatsDocument{
		FinalityProviderPkHex: fpRegistered,
		ActiveTvl:             randomAmount(r),
		TotalTvl:              randomAmount(r),
		ActiveDelegations:     r.Int63n(100),
		TotalDelegations:      r.Int63n(1000),
	}
	fpNotRegisteredStats := &model.FinalityProviderStatsDocument{
		FinalityProviderPkHex: fpNotRegistered,
		ActiveTvl:             randomAmount(r),
		TotalTvl:              randomAmount(r),
		ActiveDelegations:     r.Int63n(100),
		TotalDelegations:      r.Int63n(1000),
	}

	mockedFinalityProviderStats := &db.DbResultMap[*model.FinalityProviderStatsDocument]{
		Data:            []*model.FinalityProviderStatsDocument{fpRegisteredStats, fpNotRegisteredStats},
		PaginationToken: "abcd",
	}
	mockDB.On("FindFinalityProviderStats", mock.Anything, mock.Anything).Return(mockedFinalityProviderStats, nil)
	mockDB.On("FindFinalityProviderStatsByFinalityProviderPkHex",
		mock.Anything, mock.Anything,
	).Return([]*model.FinalityProviderStatsDocument{fpRegisteredStats}, nil)

	testServer := setupTestServer(t, &TestServerDependency{MockDbClient: mockDB, MockedFinalityProviders: fpParams})

	url := testServer.Server.URL + finalityProvidersPath
	defer testServer.Close()
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
	assert.Equal(t, 2, len(result))

	resultMap := make(map[string]services.FpDetailsPublic)
	for _, fp := range result {
		resultMap[fp.BtcPk] = fp
	}
	assert.Equal(t, int64(fpRegisteredStats.ActiveTvl), resultMap[fpRegistered].ActiveTvl)
	assert.Equal(t, int64(fpNotRegisteredStats.ActiveTvl), resultMap[fpNotRegistered].ActiveTvl)

	assert.Equal(t, "", resultMap[fpNotRegistered].Description.Moniker)

	var registeredFpParam types.FinalityProviderDetails
	for _, param := range fpParams {
		if param.BtcPk == fpRegistered {
			registeredFpParam = param
			break
		}
	}
	assert.Equal(t, registeredFpParam.Description.Moniker, resultMap[fpRegistered].Description.Moniker)
}
