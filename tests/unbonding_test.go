package tests

import (
	"bytes"
	"encoding/json"
	"io"
	"math/rand"
	"net/http"
	"testing"
	"time"

	"github.com/babylonchain/staking-queue-client/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/babylonchain/staking-api-service/internal/api"
	"github.com/babylonchain/staking-api-service/internal/api/handlers"
	"github.com/babylonchain/staking-api-service/internal/config"
	"github.com/babylonchain/staking-api-service/internal/db/model"
	"github.com/babylonchain/staking-api-service/internal/services"
	"github.com/babylonchain/staking-api-service/internal/types"
	testmock "github.com/babylonchain/staking-api-service/tests/mocks"
)

const (
	unbondingEligibilityPath = "/v1/unbonding/eligibility"
	unbondingPath            = "/v1/unbonding"
)

func TestUnbondingRequest(t *testing.T) {
	activeStakingEvent := getTestActiveStakingEvent()
	testServer := setupTestServer(t, nil)
	defer testServer.Close()

	err := sendTestMessage(testServer.Queues.ActiveStakingQueueClient, []client.ActiveStakingEvent{*activeStakingEvent})
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
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	activeStakingEvent := generateRandomActiveStakingEvents(t, r, &TestActiveEventGeneratorOpts{
		NumOfEvents:       1,
		FinalityProviders: generatePks(t, 1),
		Stakers:           generatePks(t, 1),
	})
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

func getTestActiveStakingEvent() *client.ActiveStakingEvent {
	return &client.ActiveStakingEvent{
		EventType:             client.ActiveStakingEventType,
		StakingTxHashHex:      "379155a9a081771ca64b5f73d3bf9d7611eb767d5a9f5c40aa6d769576fd35bc",
		StakerPkHex:           "b0f61bfae41af83d851a8211f82df861e93b3d39fd40a9b0e7f83bb655dad70b",
		FinalityProviderPkHex: "03d5a0bb72d71993e435d6c5a70e2aa4db500a62cfaae33c56050deefee64ec0",
		StakingValue:          100000,
		StakingStartTimestamp: 1714035159,
		StakingStartHeight:    111,
		StakingTimeLock:       10000,
		StakingOutputIndex:    1,
		StakingTxHex:          "010000000001019c867aa13a8562bc3bc53e1c59d2b4a83a9013557df575c44329d26105312d4c0000000000ffffffff030000000000000000496a476262743400b0f61bfae41af83d851a8211f82df861e93b3d39fd40a9b0e7f83bb655dad70b03d5a0bb72d71993e435d6c5a70e2aa4db500a62cfaae33c56050deefee64ec02710a0860100000000002251209e18982db19feae01b8dc208eca14b9dd7272774f0c06abc8c136b4783646fe2756a042a0100000016001403bff551edfca4d8eaaf0e5df31e391a9ed2c0360247304402204999ab524edc59ad93e6d96774f80c1dd6802f8efa53ad1f34bbcf56aa0ce889022014563a81ff980236f00c62bbf945093551f995beb806d37429729a4b998c8bd5012102b0f61bfae41af83d851a8211f82df861e93b3d39fd40a9b0e7f83bb655dad70b00000000",
		IsOverflow:            false,
	}
}

func getTestUnbondDelegationRequestPayload(stakingTxHashHex string) handlers.UnbondDelegationRequestPayload {
	return handlers.UnbondDelegationRequestPayload{
		StakingTxHashHex:         stakingTxHashHex,
		UnbondingTxHashHex:       "47ed9d80620b118c4ca558c9dd51b59fb03598eeb1674fbe57c6f7dfbbd97c7e",
		UnbondingTxHex:           "02000000000101bc35fd7695766daa405c9f5a7d76eb11769dbfd3735f4ba61c7781a0a95591370100000000ffffffff01905f01000000000022512088ae21776ea179e439e771e649a0d3b39a755e308ba0af8c71d0c2047933d6870440d780fa0e6dd463db09ad35df89c2be1eb23ad1a3d6b4dd7307894941621189a5f694d3d3eb059d7af6918695dd725c1de5abe80fb6d96f613e213af4aba303b340d0eaabc52fb941616e2da0c4a506e2694a7f82f73a4c29fef031e6a6de7982431a5eabf40ed45674c8f72d6f0c93c68dcca6706ff6009e26914f68cfa1d31411ce20b0f61bfae41af83d851a8211f82df861e93b3d39fd40a9b0e7f83bb655dad70bad2057349e985e742d5131e1e2b227b5170f6350ac2e2feb72254fcc25b3cee21a18ac2059d3532148a597a2d05c0395bf5f7176044b1cd312f37701a9b4d0aad70bc5a4ba20a5c60c2188e833d39d0fa798ab3f69aa12ed3dd2f3bad659effa252782de3c31ba20c8ccb03c379e452f10c81232b41a1ca8b63d0baf8387e57d302c987e5abb8527ba20ffeaec52a9b407b355ef6967a7ffc15fd6c3fe07de2844d61550475e7a5233e5ba539c61c050929b74c1a04954b78b4b6035e97a5e078a5a0f28ec96d547bfee9ace803ac07f9cd831c337b41bc9c6768468ea1eadb6f43285948e24ccc789629a72ddc116b83fc844969081daa83469d937e5aad55b7ae93c2df837496391c5e8d927a06700000000",
		StakerSignedSignatureHex: "d0eaabc52fb941616e2da0c4a506e2694a7f82f73a4c29fef031e6a6de7982431a5eabf40ed45674c8f72d6f0c93c68dcca6706ff6009e26914f68cfa1d31411",
	}
}

func TestProcessUnbondingStakingEvent(t *testing.T) {
	activeStakingEvent := getTestActiveStakingEvent()
	testServer := setupTestServer(t, nil)
	defer testServer.Close()

	err := sendTestMessage(testServer.Queues.ActiveStakingQueueClient, []*client.ActiveStakingEvent{activeStakingEvent})
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

	err := sendTestMessage(testServer.Queues.ActiveStakingQueueClient, []*client.ActiveStakingEvent{activeStakingEvent})
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

	err := sendTestMessage(testServer.Queues.ActiveStakingQueueClient, []*client.ActiveStakingEvent{activeStakingEvent})
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
	err = sendTestMessage(testServer.Queues.ActiveStakingQueueClient, []*client.ActiveStakingEvent{activeStakingEvent})
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

func TestUnbondingRequestValidation(t *testing.T) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	mockDB := new(testmock.DBClient)
	mockDB.On("FindDelegationByTxHashHex", mock.Anything, mock.Anything).Return(
		&model.DelegationDocument{
			State: "active",
			StakingTx: &model.TimelockTransaction{
				StartHeight: 300,
			},
		},
		nil,
	)
	testServer := setupTestServer(t, &TestServerDependency{MockDbClient: mockDB})
	defer testServer.Close()
	unbondingUrl := testServer.Server.URL + unbondingPath

	// TODO: Convert this to a table driven test, and test for all validations.
	// Test case 1: Hash from unbonding request does not match the hash on unbonding tranactions
	_, stakingTxHashHex := randomBytes(r, 32)
	_, unbondingTxHashHex := randomBytes(r, 32)
	_, unbondingTxHex, _ := generateRandomTx(r)
	_, fakeSig := randomBytes(r, 64)

	payload := handlers.UnbondDelegationRequestPayload{
		StakingTxHashHex: stakingTxHashHex,
		// unbondingTxHashHex does not match the hash of unbondingTxHex as it is
		// generated randomly
		UnbondingTxHashHex:       unbondingTxHashHex,
		UnbondingTxHex:           unbondingTxHex,
		StakerSignedSignatureHex: fakeSig,
	}

	requestBodyBytes, err := json.Marshal(payload)
	assert.NoError(t, err, "marshalling request body should not fail")

	resp, err := http.Post(unbondingUrl, "application/json", bytes.NewReader(requestBodyBytes))
	assert.NoError(t, err, "making POST request to unbonding endpoint should not fail")
	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)
	assert.NoError(t, err, "reading response body should not fail")

	var unbondingResponse api.ErrorResponse
	err = json.Unmarshal(bodyBytes, &unbondingResponse)
	assert.NoError(t, err, "unmarshalling response body should not fail")
	assert.Equal(t, types.ValidationError.String(), unbondingResponse.ErrorCode)
	assert.Equal(t, "unbonding_tx_hash_hex must match the hash calculated from the provided unbonding tx", unbondingResponse.Message)
}

func TestContentLength(t *testing.T) {
	// Setup test server with ContentLengthMiddleware
	testServer := setupTestServer(t, nil)
	defer testServer.Close()

	unbondingUrl := testServer.Server.URL + unbondingPath

	cfg, err := config.New("./config/config-test.yml")
	if err != nil {
		t.Fatal(err)
	}

	maxContentLength := cfg.Server.MaxContentLength

	// Create a payload that exceeds the max content length
	exceedingPayloadLen := maxContentLength + 1
	exceedingPayload := make([]byte, exceedingPayloadLen)
	for i := range exceedingPayload {
		exceedingPayload[i] = 'a'
	}

	// Make a POST request with the exceeding payload
	resp, err := http.Post(unbondingUrl, "application/json", bytes.NewReader(exceedingPayload))
	assert.NoError(t, err, "making POST request with exceeding payload should not fail")
	defer resp.Body.Close()

	// Check that the status code is HTTP 413 Request Entity Too Large
	assert.Equal(t, http.StatusRequestEntityTooLarge, resp.StatusCode, "expected HTTP 413 Request Entity Too Large status")

	// Test payload exactly at the limit
	exactPayload := make([]byte, maxContentLength)
	for i := range exactPayload {
		exactPayload[i] = 'a'
	}

	resp, err = http.Post(unbondingUrl, "application/json", bytes.NewReader(exactPayload))
	assert.NoError(t, err, "making POST request with exact payload should not fail")
	defer resp.Body.Close()

	assert.NotEqual(t, http.StatusRequestEntityTooLarge, resp.StatusCode, "expected status other than HTTP 413 Request Entity Too Large")

	// Test empty payload
	resp, err = http.Post(unbondingUrl, "application/json", bytes.NewReader([]byte{}))
	assert.NoError(t, err, "making POST request with empty payload should not fail")
	defer resp.Body.Close()

	assert.NotEqual(t, http.StatusRequestEntityTooLarge, resp.StatusCode, "expected status other than HTTP 413 Request Entity Too Large")

	// Create a normal payload that's below the max content length
	activeStakingEvent := getTestActiveStakingEvent()

	normalPayload := getTestUnbondDelegationRequestPayload(activeStakingEvent.StakingTxHashHex)
	requestBodyBytes, err := json.Marshal(normalPayload)
	assert.NoError(t, err, "marshalling request body should not fail")

	resp, err = http.Post(unbondingUrl, "application/json", bytes.NewReader(requestBodyBytes))
	assert.NoError(t, err, "making POST request with normal payload should not fail")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusForbidden, resp.StatusCode, "expected HTTP 403 OK status")

	// Test non-POST request
	getStakerDelegationUrl := testServer.Server.URL + stakerDelegations + "?staker_btc_pk=" + activeStakingEvent.StakerPkHex
	resp, err = http.Get(getStakerDelegationUrl)
	assert.NoError(t, err, "making GET request should not fail")
	defer resp.Body.Close()

	assert.NotEqual(t, http.StatusRequestEntityTooLarge, resp.StatusCode, "expected status other than HTTP 413 Request Entity Too Large")
}