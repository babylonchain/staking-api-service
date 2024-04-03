package tests

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/babylonchain/staking-api-service/internal/api"
	"github.com/babylonchain/staking-api-service/internal/api/handlers"
	"github.com/babylonchain/staking-api-service/internal/queue/client"
	"github.com/babylonchain/staking-api-service/internal/services"
	"github.com/babylonchain/staking-api-service/internal/types"
)

const (
	unbondingEligibilityPath = "/v1/unbonding/eligibility"
	unbondingPath            = "/v1/unbonding"
)

func TestUnbonding(t *testing.T) {
	activeStakingEvent := getTestActiveStakingEvent()
	server, queues := setupTestServer(t, nil)
	err := sendTestMessage(queues.ActiveStakingQueueClient, []client.ActiveStakingEvent{activeStakingEvent})
	require.NoError(t, err)
	defer server.Close()
	defer queues.StopReceivingMessages()

	eligibilityUrl := server.URL + unbondingEligibilityPath + "?staking_tx_hash_hex=" + activeStakingEvent.StakingTxHashHex

	time.Sleep(2 * time.Second)

	// Make a GET request to the unbonding eligibility check endpoint again
	resp, err := http.Get(eligibilityUrl)
	assert.NoError(t, err, "making GET request to unbonding eligibility check endpoint should not fail")
	defer resp.Body.Close()

	// Check that the status code is HTTP 200
	assert.Equal(t, http.StatusOK, resp.StatusCode, "expected HTTP 200 OK status")

	// Let's make a POST request to the unbonding endpoint
	unbondingUrl := server.URL + unbondingPath
	requestBody := getTestUnbondDelegationRequestPayload(activeStakingEvent.StakingTxHashHex)
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
	assert.Equal(t, "delegation not found or not eligible for unbonding", response.Message, "expected error message to be 'delegation not found or not eligible for unbonding'")

	// The state should be updated to UnbondingRequested
	getStakerDelegationUrl := server.URL + stakerDelegations + "?staker_btc_pk=" + activeStakingEvent.StakerPkHex
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
}

func TestUnbondingEligibilityWhenNoMatchingDelegation(t *testing.T) {
	activeStakingEvent := buildActiveStakingEvent(mockStakerHash, 1)
	server, queues := setupTestServer(t, nil)
	defer server.Close()
	defer queues.StopReceivingMessages()

	eligibilityUrl := server.URL + unbondingEligibilityPath + "?staking_tx_hash_hex=" + activeStakingEvent[0].StakingTxHashHex

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
		StakingTxHashHex:      "6dffa423308030246e42ee2794a35c5c2ec1ca7eb1ababb336b00ba03be525e8",
		StakerPkHex:           "4961ecc7a1e0d8563ddead4a404fa6f0249742eccba338ec9b528f3ac491b59c",
		FinalityProviderPkHex: "03d5a0bb72d71993e435d6c5a70e2aa4db500a62cfaae33c56050deefee64ec0",
		StakingValue:          75420437,
		StakingStartHeight:    111,
		StakingStartTimestamp: "StakingStartTimestamp",
		StakingTimeLock:       57587,
		StakingOutputIndex:    1,
		StakingTxHex:          "0100000000010151f86c9092cccd550603397ee98c6cb448fefe5878127207970ab9f2251f5c9c0000000000ffffffff030000000000000000496a4762627434004961ecc7a1e0d8563ddead4a404fa6f0249742eccba338ec9b528f3ac491b59c03d5a0bb72d71993e435d6c5a70e2aa4db500a62cfaae33c56050deefee64ec0e0f315d37e0400000000225120ff81e9eb925b75281a822c15e43d6ba6e28d36028c5fdcdf10a56be9d701780b001e8725010000001600146ff7d487b1e72261f7cf7e3c60daa845763ae2590247304402202b73f70d86027f1c8358169983d2fa856a55ff4550bd2e4efa984b927d66972c0220024b2c3fc8aac70acf3c1a44299ecd4df7ffbfee0822945126ba9349afa0c2070121034961ecc7a1e0d8563ddead4a404fa6f0249742eccba338ec9b528f3ac491b59c00000000",
	}
}

func getTestUnbondDelegationRequestPayload(stakingTxHashHex string) handlers.UnbondDelegationRequestPayload {
	return handlers.UnbondDelegationRequestPayload{
		StakingTxHashHex:         stakingTxHashHex,
		UnbondingTxHashHex:       "5162d2e8b701bf749cc005e9a5af656f4ddfd341332af776e743fbdc8304d67a",
		UnbondingTxHex:           "02000000000101e825e53ba00bb036b3ababb17ecac12e5c5ca39427ee426e2430803023a4ff6d0100000000ffffffff01f9bd0b0400000000225120b551ddf418a78dd6dcb20a82040953cb3b39c164498d3d265745c9c234fdaa730440b94af3ccf017804a64c8d52e307847a6826cce64f25eefc25234dd462951f0f97782c509ce98529a49b74cdc696eeaa703e45a05dbecdcd4206f703a3bf7625a40edd4ccc304384b84af81d0aec373cd23d5e05cffe3d8cfb4c92cb3bfda152a071ec4e79050023f8a4ee70b687d45f5e83bd380622dc83f2ca6439e08f315e5dace204961ecc7a1e0d8563ddead4a404fa6f0249742eccba338ec9b528f3ac491b59cad2057349e985e742d5131e1e2b227b5170f6350ac2e2feb72254fcc25b3cee21a18ac2059d3532148a597a2d05c0395bf5f7176044b1cd312f37701a9b4d0aad70bc5a4ba20a5c60c2188e833d39d0fa798ab3f69aa12ed3dd2f3bad659effa252782de3c31ba20c8ccb03c379e452f10c81232b41a1ca8b63d0baf8387e57d302c987e5abb8527ba20ffeaec52a9b407b355ef6967a7ffc15fd6c3fe07de2844d61550475e7a5233e5ba53a261c050929b74c1a04954b78b4b6035e97a5e078a5a0f28ec96d547bfee9ace803ac04155ff4d3a533921ae41eefd358ae6955a6b5e2f1302d5047548bf187733498798b0e91772d3d0b91709f6c2834f69c080728067bf246aaec186f2f7500f3bb500000000",
		StakerSignedSignatureHex: "edd4ccc304384b84af81d0aec373cd23d5e05cffe3d8cfb4c92cb3bfda152a071ec4e79050023f8a4ee70b687d45f5e83bd380622dc83f2ca6439e08f315e5da",
	}
}
