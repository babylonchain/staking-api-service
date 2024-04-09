package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/babylonchain/staking-api-service/internal/db/model"
	"github.com/babylonchain/staking-api-service/internal/types"
	"github.com/babylonchain/staking-queue-client/client"
	"github.com/stretchr/testify/assert"
)

func TestWithdrawFromActiveStaking(t *testing.T) {
	activeStakingEvent := getTestActiveStakingEvent()
	testServer := setupTestServer(t, nil)
	defer testServer.Close()
	sendTestMessage(testServer.Queues.ActiveStakingQueueClient, []client.ActiveStakingEvent{activeStakingEvent})

	// Wait for 2 seconds to make sure the message is processed
	time.Sleep(2 * time.Second)

	// Check from DB that this delegatin exist and has the state of active
	results, err := inspectDbDocuments[model.DelegationDocument](t, model.DelegationCollection)
	if err != nil {
		t.Fatalf("Failed to inspect DB documents: %v", err)
	}
	assert.Equal(t, 1, len(results), "expected 1 document in the DB")

	// Check the data
	assert.Equal(t, activeStakingEvent.StakingTxHashHex, results[0].StakingTxHashHex, "expected address to be the same")
	assert.Equal(t, types.Active, results[0].State, "expected state to be active")

	// Send the timelock expire event so that the state change to "unbonded"
	expiredEvent := client.ExpiredStakingEvent{
		EventType:        client.ExpiredStakingEventType,
		StakingTxHashHex: activeStakingEvent.StakingTxHashHex,
		TxType:           client.ActiveTxType,
	}

	sendTestMessage(testServer.Queues.ExpiredStakingQueueClient, []client.ExpiredStakingEvent{expiredEvent})
	time.Sleep(2 * time.Second)

	// Check from DB that this delegatin is in "unbonded" state
	results, err = inspectDbDocuments[model.DelegationDocument](t, model.DelegationCollection)
	if err != nil {
		t.Fatalf("Failed to inspect DB documents: %v", err)
	}
	assert.Equal(t, 1, len(results), "expected 1 document in the DB")

	// Check the data
	assert.Equal(t, activeStakingEvent.StakingTxHashHex, results[0].StakingTxHashHex, "expected address to be the same")
	assert.Equal(t, types.Unbonded, results[0].State, "expected state to be unbonded")

	// Ready for withdraw
	withdrawEvent := client.WithdrawStakingEvent{
		EventType:        client.WithdrawStakingEventType,
		StakingTxHashHex: activeStakingEvent.StakingTxHashHex,
	}

	sendTestMessage(testServer.Queues.WithdrawStakingQueueClient, []client.WithdrawStakingEvent{withdrawEvent})
	time.Sleep(2 * time.Second)

	// Check the DB, now it shall be "withdrawn" state
	results, err = inspectDbDocuments[model.DelegationDocument](t, model.DelegationCollection)
	if err != nil {
		t.Fatalf("Failed to inspect DB documents: %v", err)
	}
	assert.Equal(t, 1, len(results), "expected 1 document in the DB")

	// Check the data
	assert.Equal(t, activeStakingEvent.StakingTxHashHex, results[0].StakingTxHashHex, "expected address to be the same")
	assert.Equal(t, types.Withdrawn, results[0].State, "expected state to be unbonded")
}

func TestWithdrawFromStakingHasUnbondingRequested(t *testing.T) {
	activeStakingEvent := getTestActiveStakingEvent()
	testServer := setupTestServer(t, nil)
	defer testServer.Close()
	sendTestMessage(testServer.Queues.ActiveStakingQueueClient, []client.ActiveStakingEvent{activeStakingEvent})

	// Wait for 2 seconds to make sure the message is processed
	time.Sleep(2 * time.Second)

	// Check from DB that this delegatin exist and has the state of active
	results, err := inspectDbDocuments[model.DelegationDocument](t, model.DelegationCollection)
	if err != nil {
		t.Fatalf("Failed to inspect DB documents: %v", err)
	}
	assert.Equal(t, 1, len(results), "expected 1 document in the DB")

	// Check the data
	assert.Equal(t, activeStakingEvent.StakingTxHashHex, results[0].StakingTxHashHex, "expected address to be the same")
	assert.Equal(t, types.Active, results[0].State, "expected state to be active")

	// Let's make a POST request to the unbonding endpoint
	unbondingUrl := testServer.Server.URL + unbondingPath
	requestBody := getTestUnbondDelegationRequestPayload(activeStakingEvent.StakingTxHashHex)
	requestBodyBytes, err := json.Marshal(requestBody)
	assert.NoError(t, err, "marshalling request body should not fail")

	resp, err := http.Post(unbondingUrl, "application/json", bytes.NewReader(requestBodyBytes))
	assert.NoError(t, err, "making POST request to unbonding endpoint should not fail")
	defer resp.Body.Close()

	// Send unbonding event

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

	// Send the timelock expire event so that the state change to "unbonded"
	expiredEvent := client.ExpiredStakingEvent{
		EventType:        client.ExpiredStakingEventType,
		StakingTxHashHex: activeStakingEvent.StakingTxHashHex,
		TxType:           client.UnbondingTxType,
	}

	sendTestMessage(testServer.Queues.ExpiredStakingQueueClient, []client.ExpiredStakingEvent{expiredEvent})
	time.Sleep(2 * time.Second)

	// Check from DB that this delegatin is in "unbonded" state
	results, err = inspectDbDocuments[model.DelegationDocument](t, model.DelegationCollection)
	if err != nil {
		t.Fatalf("Failed to inspect DB documents: %v", err)
	}
	assert.Equal(t, 1, len(results), "expected 1 document in the DB")

	// Check the data
	assert.Equal(t, activeStakingEvent.StakingTxHashHex, results[0].StakingTxHashHex, "expected address to be the same")
	assert.Equal(t, types.Unbonded, results[0].State, "expected state to be unbonded")

	// Ready for withdraw
	withdrawEvent := client.WithdrawStakingEvent{
		EventType:        client.WithdrawStakingEventType,
		StakingTxHashHex: activeStakingEvent.StakingTxHashHex,
	}

	sendTestMessage(testServer.Queues.WithdrawStakingQueueClient, []client.WithdrawStakingEvent{withdrawEvent})
	time.Sleep(2 * time.Second)

	// Check the DB, now it shall be "withdrawn" state
	results, err = inspectDbDocuments[model.DelegationDocument](t, model.DelegationCollection)
	if err != nil {
		t.Fatalf("Failed to inspect DB documents: %v", err)
	}
	assert.Equal(t, 1, len(results), "expected 1 document in the DB")

	// Check the data
	assert.Equal(t, activeStakingEvent.StakingTxHashHex, results[0].StakingTxHashHex, "expected address to be the same")
	assert.Equal(t, types.Withdrawn, results[0].State, "expected state to be unbonded")
}

func TestProcessWithdrawStakingEventShouldTolerateEventMsgOutOfOrder(t *testing.T) {
	activeStakingEvent := getTestActiveStakingEvent()
	testServer := setupTestServer(t, nil)
	defer testServer.Close()
	sendTestMessage(testServer.Queues.ActiveStakingQueueClient, []client.ActiveStakingEvent{activeStakingEvent})

	// Wait for 2 seconds to make sure the message is processed
	time.Sleep(2 * time.Second)

	// Check from DB that this delegatin exist and has the state of active
	results, err := inspectDbDocuments[model.DelegationDocument](t, model.DelegationCollection)
	if err != nil {
		t.Fatalf("Failed to inspect DB documents: %v", err)
	}
	assert.Equal(t, 1, len(results), "expected 1 document in the DB")

	// Check the data
	assert.Equal(t, activeStakingEvent.StakingTxHashHex, results[0].StakingTxHashHex, "expected address to be the same")
	assert.Equal(t, types.Active, results[0].State, "expected state to be active")

	// Send the withdraw event before timelock expire event which would change the state to unbonded
	withdrawEvent := client.WithdrawStakingEvent{
		EventType:        client.WithdrawStakingEventType,
		StakingTxHashHex: activeStakingEvent.StakingTxHashHex,
	}

	sendTestMessage(testServer.Queues.WithdrawStakingQueueClient, []client.WithdrawStakingEvent{withdrawEvent})
	time.Sleep(2 * time.Second)

	// Check the DB, it should still be "active" state as the withdraw event will be requeued
	results, err = inspectDbDocuments[model.DelegationDocument](t, model.DelegationCollection)
	if err != nil {
		t.Fatalf("Failed to inspect DB documents: %v", err)
	}
	assert.Equal(t, activeStakingEvent.StakingTxHashHex, results[0].StakingTxHashHex, "expected address to be the same")
	assert.Equal(t, types.Active, results[0].State, "expected state to be active")

	// Now, send the timelock expire event so that the state change to "unbonded"
	expiredEvent := client.ExpiredStakingEvent{
		EventType:        client.ExpiredStakingEventType,
		StakingTxHashHex: activeStakingEvent.StakingTxHashHex,
		TxType:           client.ActiveTxType,
	}

	sendTestMessage(testServer.Queues.ExpiredStakingQueueClient, []client.ExpiredStakingEvent{expiredEvent})
	time.Sleep(10 * time.Second)

	// Check the DB after a while, now it shall be "withdrawn" state
	results, err = inspectDbDocuments[model.DelegationDocument](t, model.DelegationCollection)
	if err != nil {
		t.Fatalf("Failed to inspect DB documents: %v", err)
	}
	assert.Equal(t, 1, len(results), "expected 1 document in the DB")

	// Check the data
	assert.Equal(t, activeStakingEvent.StakingTxHashHex, results[0].StakingTxHashHex, "expected address to be the same")
	assert.Equal(t, types.Withdrawn, results[0].State, "expected state to be unbonded")
}
