package tests

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/babylonchain/staking-api-service/internal/api"
	"github.com/babylonchain/staking-api-service/internal/api/handlers"
	"github.com/babylonchain/staking-api-service/internal/db/model"
	"github.com/babylonchain/staking-api-service/internal/services"
	"github.com/babylonchain/staking-api-service/internal/types"
	"github.com/stretchr/testify/assert"
)

const (
	unbondingEligibilityPath = "/v1/unbonding/eligibility"
	unbondingPath            = "/v1/unbonding"
)

func TestUnbonding(t *testing.T) {
	activeStakingEvent := buildActiveStakingEvent(mockStakerHash, 1)
	testServer := setupTestServer(t, nil)
	sendTestMessage(testServer.Queues.ActiveStakingQueueClient, activeStakingEvent)
	defer testServer.Close()

	eligibilityUrl := testServer.Server.URL + unbondingEligibilityPath + "?staking_tx_hash_hex=" + activeStakingEvent[0].StakingTxHashHex

	time.Sleep(2 * time.Second)

	// Make a GET request to the unbonding eligibility check endpoint again
	resp, err := http.Get(eligibilityUrl)
	assert.NoError(t, err, "making GET request to unbonding eligibility check endpoint should not fail")
	defer resp.Body.Close()

	// Check that the status code is HTTP 200
	assert.Equal(t, http.StatusOK, resp.StatusCode, "expected HTTP 200 OK status")

	// Let's make a POST request to the unbonding endpoint
	unbondingUrl := testServer.Server.URL + unbondingPath
	requestBody := &handlers.UnbondDelegationRequestPayload{
		StakingTxHashHex:         activeStakingEvent[0].StakingTxHashHex,
		UnbondingTxHashHex:       "0x1234567890abcdef",
		UnbondingTxHex:           "0x1234567890abcdef",
		StakerSignedSignatureHex: "0x1234567890abcdef",
	}
	requestBodyBytes, err := json.Marshal(requestBody)
	assert.NoError(t, err, "marshalling request body should not fail")

	resp, err = http.Post(unbondingUrl, "application/json", bytes.NewReader(requestBodyBytes))
	assert.NoError(t, err, "making POST request to unbonding endpoint should not fail")
	defer resp.Body.Close()

	// Check that the status code is HTTP 202
	assert.Equal(t, http.StatusAccepted, resp.StatusCode, "expected HTTP 202 Accepted status")

	// Make a GET request to the unbonding eligibility check endpoint again
	resp, err = http.Get(eligibilityUrl)
	assert.NoError(t, err, "making GET request to unbonding eligibility check endpoint should not fail")
	defer resp.Body.Close()

	// Check that the status code is HTTP 403 Forbidden
	assert.Equal(t, http.StatusForbidden, resp.StatusCode, "expected HTTP 403 Forbidden status")

	// Read the response body
	bodyBytes, err := io.ReadAll(resp.Body)
	assert.NoError(t, err, "reading response body should not fail")

	var response api.ErrorResponse
	err = json.Unmarshal(bodyBytes, &response)
	assert.NoError(t, err, "unmarshalling response body should not fail")
	assert.Equal(t, "FORBIDDEN", response.ErrorCode, "expected error code to be FORBIDDEN")
	assert.Equal(t, "delegation state is not active", response.Message, "expected error message to be 'delegation state is not active'")

	// Let's make a POST request to the unbonding endpoint again
	resp, err = http.Post(unbondingUrl, "application/json", bytes.NewReader(requestBodyBytes))
	assert.NoError(t, err, "making POST request to unbonding endpoint should not fail")
	defer resp.Body.Close()

	// Check that the status code is HTTP 403 Forbidden
	assert.Equal(t, http.StatusForbidden, resp.StatusCode, "expected HTTP 403 Forbidden status")

	// Read the response body
	bodyBytes, err = io.ReadAll(resp.Body)
	assert.NoError(t, err, "reading response body should not fail")

	err = json.Unmarshal(bodyBytes, &response)
	assert.NoError(t, err, "unmarshalling response body should not fail")
	assert.Equal(t, "FORBIDDEN", response.ErrorCode, "expected error code to be FORBIDDEN")
	assert.Equal(t, "no active delegation found for unbonding request", response.Message, "expected error message to be 'no active delegation found for unbonding request'")

	// The state should be updated to UnbondingRequested
	getStakerDelegationUrl := testServer.Server.URL + stakerDelegations + "?staker_btc_pk=" + activeStakingEvent[0].StakerPkHex
	resp, err = http.Get(getStakerDelegationUrl)
	assert.NoError(t, err, "making GET request to delegations by staker pk should not fail")

	// Check that the status code is HTTP 200 OK
	assert.Equal(t, http.StatusOK, resp.StatusCode, "expected HTTP 200 OK status")

	// Read the response body
	bodyBytes, err = io.ReadAll(resp.Body)
	assert.NoError(t, err, "reading response body should not fail")

	var getStakerDelegationResponse handlers.PublicResponse[[]services.DelegationPublic]
	err = json.Unmarshal(bodyBytes, &getStakerDelegationResponse)
	assert.NoError(t, err, "unmarshalling response body should not fail")

	// Check that the response body is as expected
	assert.Equal(t, activeStakingEvent[0].StakerPkHex, getStakerDelegationResponse.Data[0].StakerPkHex, "expected response body to match")
	assert.Equal(t, types.UnbondingRequested.ToString(), getStakerDelegationResponse.Data[0].State, "state should be unbonding requested")

	// Let's inspect what's stored in the database
	results, err := inspectDbDocuments[model.UnbondingDocument](t, model.UnbondingCollection)
	assert.NoError(t, err, "failed to inspect DB documents")

	assert.Equal(t, 1, len(results), "expected 1 document in the DB")
	assert.Equal(t, activeStakingEvent[0].StakerPkHex, results[0].StakerPkHex)
	assert.Equal(t, activeStakingEvent[0].FinalityProviderPkHex, results[0].FinalityPkHex)
	assert.Equal(t, "0x1234567890abcdef", results[0].UnbondingTxSigHex)
	assert.Equal(t, "INSERTED", results[0].State)
	assert.Equal(t, activeStakingEvent[0].StakingTxHashHex, results[0].StakingTxHashHex)
	assert.Equal(t, "0x1234567890abcdef", results[0].UnbondingTxHashHex)
	assert.Equal(t, "0x1234567890abcdef", results[0].UnbondingTxHex)
	assert.Equal(t, activeStakingEvent[0].StakingTxHex, results[0].StakingTxHex)
	assert.Equal(t, activeStakingEvent[0].StakingOutputIndex, results[0].StakingOutputIndex)
	assert.Equal(t, activeStakingEvent[0].StakingTimeLock, results[0].StakingTimelock)
	assert.Equal(t, activeStakingEvent[0].StakingValue, results[0].StakingAmount)
}

func TestUnbondingEligibilityWhenNoMatchingDelegation(t *testing.T) {
	activeStakingEvent := buildActiveStakingEvent(mockStakerHash, 1)
	testServer := setupTestServer(t, nil)
	defer testServer.Close()

	eligibilityUrl := testServer.Server.URL + unbondingEligibilityPath + "?staking_tx_hash_hex=" + activeStakingEvent[0].StakingTxHashHex

	// Make a GET request to the unbonding eligibility check endpoint
	resp, err := http.Get(eligibilityUrl)
	assert.NoError(t, err, "making GET request to unbonding eligibility check endpoint should not fail")
	defer resp.Body.Close()

	// Check that the status code is HTTP 403 Forbidden
	assert.Equal(t, http.StatusForbidden, resp.StatusCode, "expected HTTP 403 Forbidden status")

	// Read the response body
	bodyBytes, err := io.ReadAll(resp.Body)
	assert.NoError(t, err, "reading response body should not fail")

	var response api.ErrorResponse
	err = json.Unmarshal(bodyBytes, &response)
	assert.NoError(t, err, "unmarshalling response body should not fail")
	assert.Equal(t, "NOT_FOUND", response.ErrorCode, "expected error code to be NOT_FOUND")
}
