package tests

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/babylonchain/staking-queue-client/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/babylonchain/staking-api-service/internal/api"
	"github.com/babylonchain/staking-api-service/internal/api/handlers"
	"github.com/babylonchain/staking-api-service/internal/db/model"
	"github.com/babylonchain/staking-api-service/internal/services"
	"github.com/babylonchain/staking-api-service/internal/types"
)

const (
	unbondingEligibilityPath = "/v1/unbonding/eligibility"
	unbondingPath            = "/v1/unbonding"
)

func TestUnbonding(t *testing.T) {
	activeStakingEvent := getTestActiveStakingEvent()
	testServer := setupTestServer(t, nil)
	err := sendTestMessage(testServer.Queues.ActiveStakingQueueClient, []client.ActiveStakingEvent{activeStakingEvent})
	defer testServer.Close()
	require.NoError(t, err)

	time.Sleep(2 * time.Second)

	eligibilityUrl := testServer.Server.URL + unbondingEligibilityPath + "?staking_tx_hash_hex=" + activeStakingEvent.StakingTxHashHex

	// Make a GET request to the unbonding eligibility check endpoint again
	resp, err := http.Get(eligibilityUrl)
	assert.NoError(t, err, "making GET request to unbonding eligibility check endpoint should not fail")
	defer resp.Body.Close()

	// Check that the status code is HTTP 200
	assert.Equal(t, http.StatusOK, resp.StatusCode, "expected HTTP 200 OK status")

	// Let's make a POST request to the unbonding endpoint
	unbondingUrl := testServer.Server.URL + unbondingPath
	requestBody := getTestUnbondDelegationRequestPayload(activeStakingEvent.StakingTxHashHex)
	requestBodyBytes, err := json.Marshal(requestBody)
	assert.NoError(t, err, "marshalling request body should not fail")

	resp, err = http.Post(unbondingUrl, "application/json", bytes.NewReader(requestBodyBytes))
	assert.NoError(t, err, "making POST request to unbonding endpoint should not fail")
	defer resp.Body.Close()

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
	assert.Equal(t, "delegation state is not active", response.Message, "expected error message to be 'no active delegation found for unbonding request'")

	// The state should be updated to UnbondingRequested
	getStakerDelegationUrl := testServer.Server.URL + stakerDelegations + "?staker_btc_pk=" + activeStakingEvent.StakerPkHex
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
	assert.Equal(t, activeStakingEvent.StakerPkHex, getStakerDelegationResponse.Data[0].StakerPkHex, "expected response body to match")
	assert.Equal(t, types.UnbondingRequested.ToString(), getStakerDelegationResponse.Data[0].State, "state should be unbonding requested")

	// Let's inspect what's stored in the database
	results, err := inspectDbDocuments[model.UnbondingDocument](t, model.UnbondingCollection)
	assert.NoError(t, err, "failed to inspect DB documents")

	assert.Equal(t, 1, len(results), "expected 1 document in the DB")
	assert.Equal(t, activeStakingEvent.StakerPkHex, results[0].StakerPkHex)
	assert.Equal(t, activeStakingEvent.FinalityProviderPkHex, results[0].FinalityPkHex)
	assert.Equal(t, requestBody.StakerSignedSignatureHex, results[0].UnbondingTxSigHex)
	assert.Equal(t, "INSERTED", results[0].State)
	assert.Equal(t, activeStakingEvent.StakingTxHashHex, results[0].StakingTxHashHex)
	assert.Equal(t, requestBody.UnbondingTxHashHex, results[0].UnbondingTxHashHex)
	assert.Equal(t, requestBody.UnbondingTxHex, results[0].UnbondingTxHex)
	assert.Equal(t, activeStakingEvent.StakingTxHex, results[0].StakingTxHex)
	assert.Equal(t, activeStakingEvent.StakingOutputIndex, results[0].StakingOutputIndex)
	assert.Equal(t, activeStakingEvent.StakingTimeLock, results[0].StakingTimelock)
	assert.Equal(t, activeStakingEvent.StakingValue, results[0].StakingAmount)
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

func getTestActiveStakingEvent() client.ActiveStakingEvent {
	return client.ActiveStakingEvent{
		EventType:             client.ActiveStakingEventType,
		StakingTxHashHex:      "e3436cff2176d636ca039dd6ba64a55d46a720b3f8d1a93f13ad522c58557af3",
		StakerPkHex:           "f9bb4e86ad4e02faab0d40bb7f7161df5fe25908ba3a80a596cd358fbe26b099",
		FinalityProviderPkHex: "03d5a0bb72d71993e435d6c5a70e2aa4db500a62cfaae33c56050deefee64ec0",
		StakingValue:          64144337,
		StakingStartTimestamp: "2024-04-03 19:34:38 +0800 CST",
		StakingStartHeight:    111,
		StakingTimeLock:       29337,
		StakingOutputIndex:    1,
		StakingTxHex:          "0100000000010181d7d9751b96db891a5d930eecbb88d283037381cbbc49c319ed38cc1a4e466f0000000000ffffffff030000000000000000496a476262743400f9bb4e86ad4e02faab0d40bb7f7161df5fe25908ba3a80a596cd358fbe26b09903d5a0bb72d71993e435d6c5a70e2aa4db500a62cfaae33c56050deefee64ec07299d1c3d203000000002251206e641710e786efc896262f3af8ae7ee46d89f392c1b15337d33eed32791a0a5c442d332601000000160014cbbbb8929921d490afe5d4f700560e69bf6fc6f00247304402202dee24a533e2cfc1e87df081c2402cc6033d172d4a9a8c3a6498a315872d1911022004d0f08b6e02bc30dcb9e70711d5ca2a4c95b2f84eb30e5ffca1552f3da35801012102f9bb4e86ad4e02faab0d40bb7f7161df5fe25908ba3a80a596cd358fbe26b09900000000",
	}
}

func getTestUnbondDelegationRequestPayload(stakingTxHashHex string) handlers.UnbondDelegationRequestPayload {
	return handlers.UnbondDelegationRequestPayload{
		StakingTxHashHex:         stakingTxHashHex,
		UnbondingTxHashHex:       "5b14b98dca6de0a921dc1abd64f8dedf9eb66e307768bf0ce9cfaa575cfcb83b",
		UnbondingTxHex:           "0200000001f37a55582c52ad133fa9d1f8b320a7465da564bad69d03ca36d67621ff6c43e30100000000ffffffff016fe3700300000000225120077251500d5425110488b9dba4c87676ed2f37c3c70bd6cff90bc1dc3d0a18d500000000",
		StakerSignedSignatureHex: "3bd94e2559ad19a7e06b884ff7e1cf46287b7213e73290f1a4c5700ee660daf29d0da9a74c65f21751e97229d4e82e26117eee5c48c5ecf3a3be697ecee0be36",
	}
}
