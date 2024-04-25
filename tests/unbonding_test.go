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

func TestUnbondingRequest(t *testing.T) {
	activeStakingEvent := getTestActiveStakingEvent()
	testServer := setupTestServer(t, nil)
	defer testServer.Close()

	err := sendTestMessage(testServer.Queues.ActiveStakingQueueClient, []client.ActiveStakingEvent{activeStakingEvent})
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
	defer resp.Body.Close()
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

func TestUnbondingRequestEligibilityWhenNoMatchingDelegation(t *testing.T) {
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
		StakingTxHashHex:      "8063a57b1e46383e1683a33ced753bcaa056cfd10a98c3f4a5040e5b5f8d9948",
		StakerPkHex:           "68d4cd21b679dc4c16bdddaed7d627a98e8b6adbf113f20e272d454f9404527a",
		FinalityProviderPkHex: "03d5a0bb72d71993e435d6c5a70e2aa4db500a62cfaae33c56050deefee64ec0",
		StakingValue:          100000,
		StakingStartTimestamp: 1714035159,
		StakingStartHeight:    111,
		StakingTimeLock:       10000,
		StakingOutputIndex:    1,
		StakingTxHex:          "010000000001015cf30114f520715be461e0b3baa4ef2a20cf6e3a0e71013b3b1139b5804fcb400000000000ffffffff030000000000000000496a47626274340068d4cd21b679dc4c16bdddaed7d627a98e8b6adbf113f20e272d454f9404527a03d5a0bb72d71993e435d6c5a70e2aa4db500a62cfaae33c56050deefee64ec02710a086010000000000225120be5fb13e25024c8be2ffe927e2279ae2bcbce1fc31b0f74a654a9bd41b9d1168756a042a01000000160014002f76d7f66724b008380a5ce8470a7bd51067ed02473044022022836ac3c693dd8ea03c997e049f9c8d777cfc4c74bdf490f24f85e64a745112022006ab077a993bcc25e699fe2467f171a1d8950e0fe3de7dcaa52d51c076e4923701210368d4cd21b679dc4c16bdddaed7d627a98e8b6adbf113f20e272d454f9404527a00000000",
	}
}

func getTestUnbondDelegationRequestPayload(stakingTxHashHex string) handlers.UnbondDelegationRequestPayload {
	return handlers.UnbondDelegationRequestPayload{
		StakingTxHashHex:         stakingTxHashHex,
		UnbondingTxHashHex:       "19550a9efa5ae2ace1204ba255ad197c582b9fb01eaf6ffbf71086e1f7244ad2",
		UnbondingTxHex:           "0200000000010148998d5f5b0e04a5f4c3980ad1cf56a0ca3b75ed3ca383163e38461e7ba563800100000000ffffffff01905f010000000000225120633c8ac68993f17d28feca0fc7d00d166f11c6873aec41b3ff48227432ebaef60440bb9593a6ff4a62ca026dbf92a3b8802c94630afb9c07cb29f7863980a3ae760df554b817fc7242f3e1cac8ab897e06d39c5051f86808e84661d1a8da6e4ba0e4404079a1e119a9880e4869f099bc0a32efeaa0e07cbb4b9a44f747e62c18848de3f57b28c40d93aa59159300dd0ee34f13579fc0315955521526a9db520b3615a4ce2068d4cd21b679dc4c16bdddaed7d627a98e8b6adbf113f20e272d454f9404527aad2057349e985e742d5131e1e2b227b5170f6350ac2e2feb72254fcc25b3cee21a18ac2059d3532148a597a2d05c0395bf5f7176044b1cd312f37701a9b4d0aad70bc5a4ba20a5c60c2188e833d39d0fa798ab3f69aa12ed3dd2f3bad659effa252782de3c31ba20c8ccb03c379e452f10c81232b41a1ca8b63d0baf8387e57d302c987e5abb8527ba20ffeaec52a9b407b355ef6967a7ffc15fd6c3fe07de2844d61550475e7a5233e5ba53a261c150929b74c1a04954b78b4b6035e97a5e078a5a0f28ec96d547bfee9ace803ac0d1e0e9597664a20ba7f196dd619e80ee77f2595e51fe1638971af894a18b5f492926b5110525ca103c70b6ce5cd140a6c39a94dfdaf1a416b54478be11c159fa00000000",
		StakerSignedSignatureHex: "4079a1e119a9880e4869f099bc0a32efeaa0e07cbb4b9a44f747e62c18848de3f57b28c40d93aa59159300dd0ee34f13579fc0315955521526a9db520b3615a4",
	}
}

func TestProcessUnbondingStakingEvent(t *testing.T) {
	activeStakingEvent := getTestActiveStakingEvent()
	testServer := setupTestServer(t, nil)
	defer testServer.Close()

	err := sendTestMessage(testServer.Queues.ActiveStakingQueueClient, []client.ActiveStakingEvent{activeStakingEvent})
	require.NoError(t, err)

	time.Sleep(2 * time.Second)

	// Let's make a POST request to the unbonding endpoint
	unbondingUrl := testServer.Server.URL + unbondingPath
	requestBody := getTestUnbondDelegationRequestPayload(activeStakingEvent.StakingTxHashHex)
	requestBodyBytes, err := json.Marshal(requestBody)
	assert.NoError(t, err, "marshalling request body should not fail")

	resp, err := http.Post(unbondingUrl, "application/json", bytes.NewReader(requestBodyBytes))
	assert.NoError(t, err, "making POST request to unbonding endpoint should not fail")
	defer resp.Body.Close()

	// Let's inspect what's stored in the database
	results, err := inspectDbDocuments[model.UnbondingDocument](t, model.UnbondingCollection)
	assert.NoError(t, err, "failed to inspect DB documents")

	assert.Equal(t, 1, len(results), "expected 1 document in the DB")
	assert.Equal(t, "INSERTED", results[0].State)
	assert.Equal(t, activeStakingEvent.StakingTxHex, results[0].StakingTxHex)

	// Let's send an unbonding event
	unbondingEvent := client.UnbondingStakingEvent{
		EventType:               client.UnbondingStakingEventType,
		StakingTxHashHex:        requestBody.StakingTxHashHex,
		UnbondingTxHashHex:      requestBody.UnbondingTxHashHex,
		UnbondingTxHex:          requestBody.UnbondingTxHex,
		UnbondingTimeLock:       10,
		UnbondingStartTimestamp: time.Now().Unix(),
		UnbondingStartHeight:    activeStakingEvent.StakingStartHeight + 100,
		UnbondingOutputIndex:    1,
	}

	sendTestMessage(testServer.Queues.UnbondingStakingQueueClient, []client.UnbondingStakingEvent{unbondingEvent})
	time.Sleep(2 * time.Second)

	// Let's GET the delegation from API
	getStakerDelegationUrl := testServer.Server.URL + stakerDelegations + "?staker_btc_pk=" + activeStakingEvent.StakerPkHex
	resp, err = http.Get(getStakerDelegationUrl)
	assert.NoError(t, err, "making GET request to delegations by staker pk should not fail")
	defer resp.Body.Close()
	// Check that the status code is HTTP 200 OK
	assert.Equal(t, http.StatusOK, resp.StatusCode, "expected HTTP 200 OK status")

	// Read the response body
	bodyBytes, err := io.ReadAll(resp.Body)
	assert.NoError(t, err, "reading response body should not fail")

	var getStakerDelegationResponse handlers.PublicResponse[[]services.DelegationPublic]
	err = json.Unmarshal(bodyBytes, &getStakerDelegationResponse)
	assert.NoError(t, err, "unmarshalling response body should not fail")

	// Check that the response body is as expected
	assert.Equal(t, 1, len(getStakerDelegationResponse.Data), "expected 1 delegation in the response")
	assert.Equal(t, activeStakingEvent.StakerPkHex, getStakerDelegationResponse.Data[0].StakerPkHex, "expected response body to match")
	assert.Equal(t, types.Unbonding.ToString(), getStakerDelegationResponse.Data[0].State, "state should be unbonding")
	// Make sure the unbonding tx exist in the response body
	assert.NotNil(t, getStakerDelegationResponse.Data[0].UnbondingTx, "expected unbonding tx to be present in the response body")
	assert.Equal(t, unbondingEvent.UnbondingTxHex, getStakerDelegationResponse.Data[0].UnbondingTx.TxHex, "expected unbonding tx to match")

	_, err = time.Parse(time.RFC3339, getStakerDelegationResponse.Data[0].UnbondingTx.StartTimestamp)
	assert.NoError(t, err, "expected timestamp to be in RFC3339 format")

	// Let's also fetch the DB to make sure the expired check is processed
	timeLockResults, err := inspectDbDocuments[model.TimeLockDocument](t, model.TimeLockCollection)
	assert.NoError(t, err, "failed to inspect DB documents")

	assert.Equal(t, 2, len(timeLockResults), "expected 2 document in the DB")
	// The first one is from the
	assert.Equal(t, activeStakingEvent.StakingTxHashHex, timeLockResults[0].StakingTxHashHex)
	assert.Equal(t, types.ActiveTxType.ToString(), timeLockResults[0].TxType)
	// Point to the same staking tx hash
	assert.Equal(t, activeStakingEvent.StakingTxHashHex, timeLockResults[1].StakingTxHashHex)
	assert.Equal(t, requestBody.StakingTxHashHex, timeLockResults[1].StakingTxHashHex)
	assert.Equal(t, types.UnbondingTxType.ToString(), timeLockResults[1].TxType)
}

func TestProcessUnbondingStakingEventDuringBootstrap(t *testing.T) {
	activeStakingEvent := getTestActiveStakingEvent()
	testServer := setupTestServer(t, nil)
	defer testServer.Close()

	err := sendTestMessage(testServer.Queues.ActiveStakingQueueClient, []client.ActiveStakingEvent{activeStakingEvent})
	require.NoError(t, err)

	time.Sleep(2 * time.Second)

	// We generate the necessary unbonding request payload, but we not sending it to the unbonding endpoint
	// Instead, we send the unbonding event directly to simulate the case where our system is bootstrapping
	// which means we have the unbonding data but not the unbonding request
	requestBody := getTestUnbondDelegationRequestPayload(activeStakingEvent.StakingTxHashHex)
	unbondingEvent := client.UnbondingStakingEvent{
		EventType:               client.UnbondingStakingEventType,
		StakingTxHashHex:        requestBody.StakingTxHashHex,
		UnbondingTxHashHex:      requestBody.UnbondingTxHashHex,
		UnbondingTxHex:          requestBody.UnbondingTxHex,
		UnbondingTimeLock:       10,
		UnbondingStartTimestamp: time.Now().Unix(),
		UnbondingStartHeight:    activeStakingEvent.StakingStartHeight + 100,
		UnbondingOutputIndex:    1,
	}

	sendTestMessage(testServer.Queues.UnbondingStakingQueueClient, []client.UnbondingStakingEvent{unbondingEvent})
	time.Sleep(2 * time.Second)

	// Let's GET the delegation from API
	getStakerDelegationUrl := testServer.Server.URL + stakerDelegations + "?staker_btc_pk=" + activeStakingEvent.StakerPkHex
	resp, err := http.Get(getStakerDelegationUrl)
	assert.NoError(t, err, "making GET request to delegations by staker pk should not fail")
	defer resp.Body.Close()

	// Check that the status code is HTTP 200 OK
	assert.Equal(t, http.StatusOK, resp.StatusCode, "expected HTTP 200 OK status")

	// Read the response body
	bodyBytes, err := io.ReadAll(resp.Body)
	assert.NoError(t, err, "reading response body should not fail")

	var getStakerDelegationResponse handlers.PublicResponse[[]services.DelegationPublic]
	err = json.Unmarshal(bodyBytes, &getStakerDelegationResponse)
	assert.NoError(t, err, "unmarshalling response body should not fail")

	// Check that the response body is as expected
	assert.Equal(t, 1, len(getStakerDelegationResponse.Data), "expected 1 delegation in the response")
	assert.Equal(t, activeStakingEvent.StakerPkHex, getStakerDelegationResponse.Data[0].StakerPkHex, "expected response body to match")
	assert.Equal(t, types.Unbonding.ToString(), getStakerDelegationResponse.Data[0].State, "state should be unbonding")
	// Make sure the unbonding tx exist in the response body
	assert.NotNil(t, getStakerDelegationResponse.Data[0].UnbondingTx, "expected unbonding tx to be present in the response body")
	assert.Equal(t, unbondingEvent.UnbondingTxHex, getStakerDelegationResponse.Data[0].UnbondingTx.TxHex, "expected unbonding tx to match")

	// Let's also fetch the DB to make sure the expired check is processed
	timeLockResults, err := inspectDbDocuments[model.TimeLockDocument](t, model.TimeLockCollection)
	assert.NoError(t, err, "failed to inspect DB documents")

	assert.Equal(t, 2, len(timeLockResults), "expected 2 document in the DB")
	// The first one is from the
	assert.Equal(t, activeStakingEvent.StakingTxHashHex, timeLockResults[0].StakingTxHashHex)
	assert.Equal(t, types.ActiveTxType.ToString(), timeLockResults[0].TxType)
	// Point to the same staking tx hash
	assert.Equal(t, activeStakingEvent.StakingTxHashHex, timeLockResults[1].StakingTxHashHex)
	assert.Equal(t, requestBody.StakingTxHashHex, timeLockResults[1].StakingTxHashHex)
	assert.Equal(t, types.UnbondingTxType.ToString(), timeLockResults[1].TxType)
}

func TestShouldIgnoreOutdatedUnbondingEvent(t *testing.T) {
	activeStakingEvent := getTestActiveStakingEvent()
	testServer := setupTestServer(t, nil)
	defer testServer.Close()

	err := sendTestMessage(testServer.Queues.ActiveStakingQueueClient, []client.ActiveStakingEvent{activeStakingEvent})
	require.NoError(t, err)

	time.Sleep(2 * time.Second)

	requestBody := getTestUnbondDelegationRequestPayload(activeStakingEvent.StakingTxHashHex)
	unbondingEvent := client.UnbondingStakingEvent{
		EventType:               client.UnbondingStakingEventType,
		StakingTxHashHex:        requestBody.StakingTxHashHex,
		UnbondingTxHashHex:      requestBody.UnbondingTxHashHex,
		UnbondingTxHex:          requestBody.UnbondingTxHex,
		UnbondingTimeLock:       10,
		UnbondingStartTimestamp: time.Now().Unix(),
		UnbondingStartHeight:    activeStakingEvent.StakingStartHeight + 100,
		UnbondingOutputIndex:    1,
	}

	sendTestMessage(testServer.Queues.UnbondingStakingQueueClient, []client.UnbondingStakingEvent{unbondingEvent})
	time.Sleep(2 * time.Second)

	// Let's GET the delegation from API
	getStakerDelegationUrl := testServer.Server.URL + stakerDelegations + "?staker_btc_pk=" + activeStakingEvent.StakerPkHex
	resp, err := http.Get(getStakerDelegationUrl)
	assert.NoError(t, err, "making GET request to delegations by staker pk should not fail")
	defer resp.Body.Close()

	// Check that the status code is HTTP 200 OK
	assert.Equal(t, http.StatusOK, resp.StatusCode, "expected HTTP 200 OK status")

	// Read the response body
	bodyBytes, err := io.ReadAll(resp.Body)
	assert.NoError(t, err, "reading response body should not fail")

	var getStakerDelegationResponse handlers.PublicResponse[[]services.DelegationPublic]
	err = json.Unmarshal(bodyBytes, &getStakerDelegationResponse)
	assert.NoError(t, err, "unmarshalling response body should not fail")

	// Check that the response body is as expected
	assert.Equal(t, 1, len(getStakerDelegationResponse.Data), "expected 1 delegation in the response")
	assert.Equal(t, activeStakingEvent.StakerPkHex, getStakerDelegationResponse.Data[0].StakerPkHex, "expected response body to match")
	assert.Equal(t, types.Unbonding.ToString(), getStakerDelegationResponse.Data[0].State, "state should be unbonding")
	// Make sure the unbonding tx exist in the response body
	assert.NotNil(t, getStakerDelegationResponse.Data[0].UnbondingTx, "expected unbonding tx to be present in the response body")
	assert.Equal(t, unbondingEvent.UnbondingTxHex, getStakerDelegationResponse.Data[0].UnbondingTx.TxHex, "expected unbonding tx to match")

	// Let's also fetch the DB to make sure the expired check is processed
	timeLockResults, err := inspectDbDocuments[model.TimeLockDocument](t, model.TimeLockCollection)
	assert.NoError(t, err, "failed to inspect DB documents")

	assert.Equal(t, 2, len(timeLockResults), "expected 2 document in the DB")
	// The first one is from the
	assert.Equal(t, activeStakingEvent.StakingTxHashHex, timeLockResults[0].StakingTxHashHex)
	assert.Equal(t, types.ActiveTxType.ToString(), timeLockResults[0].TxType)
	// Point to the same staking tx hash
	assert.Equal(t, activeStakingEvent.StakingTxHashHex, timeLockResults[1].StakingTxHashHex)
	assert.Equal(t, requestBody.StakingTxHashHex, timeLockResults[1].StakingTxHashHex)
	assert.Equal(t, types.UnbondingTxType.ToString(), timeLockResults[1].TxType)

	// Let's send an outdated unbonding event
	sendTestMessage(testServer.Queues.UnbondingStakingQueueClient, []client.UnbondingStakingEvent{unbondingEvent})
	time.Sleep(2 * time.Second)

	// Fetch from the expire checker to make sure we only processed the unbonding event once
	timeLockResults, err = inspectDbDocuments[model.TimeLockDocument](t, model.TimeLockCollection)
	assert.NoError(t, err, "failed to inspect DB documents")

	// Should still be 2
	assert.Equal(t, 2, len(timeLockResults), "expected 2 document in the DB")
}

func TestProcessUnbondingStakingEventShouldTolerateEventMsgOutOfOrder(t *testing.T) {
	testServer := setupTestServer(t, nil)
	defer testServer.Close()

	// We generate the active event, but not sending it yet. we will send it after sending the unbonding event
	activeStakingEvent := getTestActiveStakingEvent()
	requestBody := getTestUnbondDelegationRequestPayload(activeStakingEvent.StakingTxHashHex)
	unbondingEvent := client.UnbondingStakingEvent{
		EventType:               client.UnbondingStakingEventType,
		StakingTxHashHex:        requestBody.StakingTxHashHex,
		UnbondingTxHashHex:      requestBody.UnbondingTxHashHex,
		UnbondingTxHex:          requestBody.UnbondingTxHex,
		UnbondingTimeLock:       10,
		UnbondingStartTimestamp: time.Now().Unix(),
		UnbondingStartHeight:    activeStakingEvent.StakingStartHeight + 100,
		UnbondingOutputIndex:    1,
	}

	sendTestMessage(testServer.Queues.UnbondingStakingQueueClient, []client.UnbondingStakingEvent{unbondingEvent})
	time.Sleep(2 * time.Second)
	// Check DB, there should be no unbonding document
	results, err := inspectDbDocuments[model.UnbondingDocument](t, model.UnbondingCollection)
	assert.NoError(t, err, "failed to inspect DB documents")
	assert.Empty(t, results, "expected no unbonding document in the DB")

	// Send the active event
	err = sendTestMessage(testServer.Queues.ActiveStakingQueueClient, []client.ActiveStakingEvent{activeStakingEvent})
	require.NoError(t, err)

	time.Sleep(10 * time.Second)

	// Let's GET the delegation from API
	getStakerDelegationUrl := testServer.Server.URL + stakerDelegations + "?staker_btc_pk=" + activeStakingEvent.StakerPkHex
	resp, err := http.Get(getStakerDelegationUrl)
	assert.NoError(t, err, "making GET request to delegations by staker pk should not fail")
	defer resp.Body.Close()

	// Check that the status code is HTTP 200 OK
	assert.Equal(t, http.StatusOK, resp.StatusCode, "expected HTTP 200 OK status")

	// Read the response body
	bodyBytes, err := io.ReadAll(resp.Body)
	assert.NoError(t, err, "reading response body should not fail")

	var getStakerDelegationResponse handlers.PublicResponse[[]services.DelegationPublic]
	err = json.Unmarshal(bodyBytes, &getStakerDelegationResponse)
	assert.NoError(t, err, "unmarshalling response body should not fail")

	// Check that the response body is as expected
	assert.Equal(t, 1, len(getStakerDelegationResponse.Data), "expected 1 delegation in the response")
	assert.Equal(t, types.Unbonding.ToString(), getStakerDelegationResponse.Data[0].State, "state should be unbonding")
	// Make sure the unbonding tx exist in the response body
	assert.NotNil(t, getStakerDelegationResponse.Data[0].UnbondingTx, "expected unbonding tx to be present in the response body")
	assert.Equal(t, unbondingEvent.UnbondingTxHex, getStakerDelegationResponse.Data[0].UnbondingTx.TxHex, "expected unbonding tx to match")

	// Let's also fetch the DB to make sure the expired check is processed
	timeLockResults, err := inspectDbDocuments[model.TimeLockDocument](t, model.TimeLockCollection)
	assert.NoError(t, err, "failed to inspect DB documents")

	assert.Equal(t, 2, len(timeLockResults), "expected 2 document in the DB")
	// The first one is from the
	assert.Equal(t, activeStakingEvent.StakingTxHashHex, timeLockResults[0].StakingTxHashHex)
	assert.Equal(t, types.ActiveTxType.ToString(), timeLockResults[0].TxType)
	// Point to the same staking tx hash
	assert.Equal(t, activeStakingEvent.StakingTxHashHex, timeLockResults[1].StakingTxHashHex)
	assert.Equal(t, requestBody.StakingTxHashHex, timeLockResults[1].StakingTxHashHex)
	assert.Equal(t, types.UnbondingTxType.ToString(), timeLockResults[1].TxType)
}
