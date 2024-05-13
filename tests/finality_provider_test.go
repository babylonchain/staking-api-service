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
	"github.com/babylonchain/staking-api-service/internal/config"
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

func FuzzGetFinalityProviderShouldReturnAllRegisteredFps(f *testing.F) {
	attachRandomSeedsToFuzzer(f, 100)
	f.Fuzz(func(t *testing.T, seed int64) {
		r := rand.New(rand.NewSource(seed))
		fpParams, registeredFpsStats, notRegisteredFpsStats := setUpFinalityProvidersStatsDataSet(t, r)

		mockDB := new(testmock.DBClient)
		mockDB.On("FindFinalityProviderStatsByFinalityProviderPkHex",
			mock.Anything, mock.Anything,
		).Return(registeredFpsStats, nil)

		mockedFinalityProviderStats := &db.DbResultMap[*model.FinalityProviderStatsDocument]{
			Data:            append(registeredFpsStats, notRegisteredFpsStats...),
			PaginationToken: "",
		}
		mockDB.On("FindFinalityProviderStats", mock.Anything, mock.Anything).Return(mockedFinalityProviderStats, nil)

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

		assert.NotEmptyf(t, result, "expected response body to be non-empty")
		// We expect all registered finality providers to be returned, plus the one that is not registered
		var fpParamsWithStakingMap = make(map[string]bool)
		for _, fp := range fpParams {
			found := false
			for _, fpStat := range registeredFpsStats {
				if fp.BtcPk == fpStat.FinalityProviderPkHex {
					found = true
					break
				}
			}
			fpParamsWithStakingMap[fp.BtcPk] = found
		}
		assert.Equal(t, len(fpParams)+len(notRegisteredFpsStats), len(result))

		resultMap := make(map[string]services.FpDetailsPublic)
		for _, fp := range result {
			resultMap[fp.BtcPk] = fp
		}

		// Check all the registered finality providers should apprear in the response
		for _, f := range fpParams {
			assert.Equal(t, f.Description.Moniker, resultMap[f.BtcPk].Description.Moniker)
			assert.Equal(t, f.Commission, resultMap[f.BtcPk].Commission)
			// Check that the stats are correct for the registered finality providers without any delegations
			if fpParamsWithStakingMap[f.BtcPk] == false {
				assert.Equal(t, int64(0), resultMap[f.BtcPk].ActiveTvl)
				assert.Equal(t, int64(0), resultMap[f.BtcPk].TotalTvl)
				assert.Equal(t, int64(0), resultMap[f.BtcPk].ActiveDelegations)
				assert.Equal(t, int64(0), resultMap[f.BtcPk].TotalDelegations)
			} else {
				assert.NotZero(t, resultMap[f.BtcPk].ActiveTvl)
				assert.NotZero(t, resultMap[f.BtcPk].TotalTvl)
				assert.NotZero(t, resultMap[f.BtcPk].ActiveDelegations)
				assert.NotZero(t, resultMap[f.BtcPk].TotalDelegations)
			}
		}
		for _, f := range notRegisteredFpsStats {
			assert.Equal(t, "", resultMap[f.FinalityProviderPkHex].Description.Moniker)
		}
	})
}

func FuzzTestGetFinalityProviderWithPaginationResponse(f *testing.F) {
	attachRandomSeedsToFuzzer(f, 3)
	f.Fuzz(func(t *testing.T, seed int64) {
		r := rand.New(rand.NewSource(seed))
		opts := &TestActiveEventGeneratorOpts{
			NumOfEvents:     20,
			NumberOfFps:     10,
			NumberOfStakers: randomPositiveInt(r, 10),
		}
		activeStakingEvents := generateRandomActiveStakingEvents(t, r, opts)
		cfg, err := config.New("./config/config-test.yml")
		if err != nil {
			t.Fatalf("Failed to load test config: %v", err)
		}
		cfg.Db.MaxPaginationLimit = 2

		testServer := setupTestServer(t, &TestServerDependency{ConfigOverrides: cfg})
		defer testServer.Close()
		sendTestMessage(testServer.Queues.ActiveStakingQueueClient, activeStakingEvents)
		time.Sleep(10 * time.Second)

		var paginationKey string
		var allDataCollected []services.FpDetailsPublic
		var atLeastOnePage bool
		// Test the API
		for {
			url := testServer.Server.URL + finalityProvidersPath + "?pagination_key=" + paginationKey
			resp, err := http.Get(url)
			assert.NoError(t, err, "making GET request to finality providers endpoint should not fail")
			assert.Equal(t, http.StatusOK, resp.StatusCode, "expected HTTP 200 OK status")
			bodyBytes, err := io.ReadAll(resp.Body)
			assert.NoError(t, err, "reading response body should not fail")
			var response handlers.PublicResponse[[]services.FpDetailsPublic]
			err = json.Unmarshal(bodyBytes, &response)
			assert.NoError(t, err, "unmarshalling response body should not fail")

			// Check that the response body is as expected
			assert.NotEmptyf(t, response.Data, "expected response body to have data")
			allDataCollected = append(allDataCollected, response.Data...)
			if response.Pagination.NextKey != "" {
				atLeastOnePage = true
				paginationKey = response.Pagination.NextKey
			} else {
				break
			}
		}

		assert.True(t, atLeastOnePage, "expected at least one page")
		for i := 0; i < len(allDataCollected)-1; i++ {
			assert.True(t, allDataCollected[i].ActiveTvl >= allDataCollected[i+1].ActiveTvl)
		}
	})
}

func FuzzGetFinalityProviderShouldNotReturnRegisteredFpWithoutStakingForPaginatedDbResponse(f *testing.F) {
	attachRandomSeedsToFuzzer(f, 100)
	f.Fuzz(func(t *testing.T, seed int64) {
		r := rand.New(rand.NewSource(seed))
		fpParams, registeredFpsStats, notRegisteredFpsStats := setUpFinalityProvidersStatsDataSet(t, r)

		mockDB := new(testmock.DBClient)
		mockDB.On("FindFinalityProviderStatsByFinalityProviderPkHex",
			mock.Anything, mock.Anything,
		).Return(registeredFpsStats, nil)

		mockedFinalityProviderStats := &db.DbResultMap[*model.FinalityProviderStatsDocument]{
			Data:            append(registeredFpsStats, notRegisteredFpsStats...),
			PaginationToken: "abcd",
		}
		mockDB.On("FindFinalityProviderStats", mock.Anything, mock.Anything).Return(mockedFinalityProviderStats, nil)

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

		var fpParamWithStakingMap []string
		for _, fp := range fpParams {
			for _, fpStat := range registeredFpsStats {
				if fp.BtcPk == fpStat.FinalityProviderPkHex {
					fpParamWithStakingMap = append(fpParamWithStakingMap, fp.BtcPk)
					break
				}
			}
		}

		assert.Equal(t, len(registeredFpsStats)+len(notRegisteredFpsStats), len(result))
		assert.Less(t, len(fpParamWithStakingMap), len(fpParams))
	})
}

func generateFinalityProviderStatsDocument(r *rand.Rand, pk string) *model.FinalityProviderStatsDocument {
	return &model.FinalityProviderStatsDocument{
		FinalityProviderPkHex: pk,
		ActiveTvl:             randomAmount(r),
		TotalTvl:              randomAmount(r),
		ActiveDelegations:     r.Int63n(100) + 1,
		TotalDelegations:      r.Int63n(1000) + 1,
	}
}

func setUpFinalityProvidersStatsDataSet(t *testing.T, r *rand.Rand) ([]types.FinalityProviderDetails, []*model.FinalityProviderStatsDocument, []*model.FinalityProviderStatsDocument) {
	numOfRegisterFps := r.Uint64()%10 + 1
	fpParams := generateRandomFinalityProviderDetail(t, r, numOfRegisterFps)

	// Generate a set of registered finality providers
	var registeredFpsStats []*model.FinalityProviderStatsDocument
	for i := uint64(0); i < r.Uint64()%numOfRegisterFps; i++ {
		fpStats := generateFinalityProviderStatsDocument(r, fpParams[i].BtcPk)
		registeredFpsStats = append(registeredFpsStats, fpStats)
	}

	var notRegisteredFpsStats []*model.FinalityProviderStatsDocument
	numOfNotRegisteredFps := r.Uint64()%10 + 1
	for i := uint64(0); i < numOfNotRegisteredFps; i++ {
		fpNotRegisteredPk, err := randomPk()
		assert.NoError(t, err, "generating random public key should not fail")

		stats := generateFinalityProviderStatsDocument(r, fpNotRegisteredPk)
		notRegisteredFpsStats = append(notRegisteredFpsStats, stats)
	}
	assert.LessOrEqual(t, len(registeredFpsStats), len(fpParams))

	return fpParams, registeredFpsStats, notRegisteredFpsStats
}
