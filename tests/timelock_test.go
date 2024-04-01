package tests

import (
	"testing"
	"time"

	"github.com/babylonchain/staking-api-service/internal/db/model"
	"github.com/stretchr/testify/assert"
)

func TestSaveTimelock(t *testing.T) {
	// Inject random data
	activeStakingEvent := buildActiveStakingEvent("0x1234567890abcdef", 1)
	server, queues := setupTestServer(t, nil)
	sendTestMessage(queues.ActiveStakingQueueClient, activeStakingEvent)
	defer server.Close()
	defer queues.StopReceivingMessages()

	// Wait for 2 seconds to make sure the message is processed
	time.Sleep(2 * time.Second)
	// Check from DB if the data is saved
	results, err := inspectDbDocuments[model.TimeLockDocument](t, model.TimeLockCollection)
	if err != nil {
		t.Fatalf("Failed to inspect DB documents: %v", err)
	}
	assert.Equal(t, 1, len(results), "expected 1 document in the DB")

	// Check the data
	assert.Equal(t, activeStakingEvent[0].StakingTxHashHex, results[0].StakingTxHashHex, "expected address to be the same")

	expectedExpireHeight := activeStakingEvent[0].StakingStartHeight + activeStakingEvent[0].StakingTimeLock
	assert.Equal(t, expectedExpireHeight, results[0].ExpireHeight, "expected address to be the same")
}

// Note: We allow duplication to happen in the expire checker. It's the responsibility of the expired event receiver to handle the duplication.
// Two different type of duplication could happen: 1. Duplicated txHashHex but with different expireHeight. 2. Duplicated txHashHex with the same expireHeight.
func TestSaveTimelockWithDuplicates(t *testing.T) {
	// Inject random data
	activeStakingEvent := buildActiveStakingEvent("0x1234567890abcdef", 1)
	server, queues := setupTestServer(t, nil)
	sendTestMessage(queues.ActiveStakingQueueClient, activeStakingEvent)
	// Send again
	sendTestMessage(queues.ActiveStakingQueueClient, activeStakingEvent)
	defer server.Close()
	defer queues.StopReceivingMessages()

	// Wait for 2 seconds to make sure the message is processed
	time.Sleep(5 * time.Second)
	// Check from DB if the data is saved
	results, err := inspectDbDocuments[model.TimeLockDocument](t, model.TimeLockCollection)
	if err != nil {
		t.Fatalf("Failed to inspect DB documents: %v", err)
	}
	assert.Equal(t, 2, len(results), "expected 2 document in the DB")

	// Check the data
	assert.Equal(t, activeStakingEvent[0].StakingTxHashHex, results[0].StakingTxHashHex, "expected address to be the same")
	assert.Equal(t, activeStakingEvent[0].StakingTxHashHex, results[1].StakingTxHashHex, "expected address to be the same")
	expectedExpireHeight := activeStakingEvent[0].StakingStartHeight + activeStakingEvent[0].StakingTimeLock
	assert.Equal(t, expectedExpireHeight, results[0].ExpireHeight, "expected address to be the same")
	assert.Equal(t, expectedExpireHeight, results[1].ExpireHeight, "expected address to be the same")

	// Now, let's inject the same data but with different expireHeight
	eventWithDifferentExpireHeight := buildActiveStakingEvent("0x1234567890abcdef", 1)
	expectedExpireHeight = eventWithDifferentExpireHeight[0].StakingStartHeight + eventWithDifferentExpireHeight[0].StakingTimeLock

	sendTestMessage(queues.ActiveStakingQueueClient, eventWithDifferentExpireHeight)
	time.Sleep(2 * time.Second)

	// Check from DB if the data is saved
	results, err = inspectDbDocuments[model.TimeLockDocument](t, model.TimeLockCollection)
	if err != nil {
		t.Fatalf("Failed to inspect DB documents: %v", err)
	}
	assert.Equal(t, 3, len(results), "expected 3 document in the DB")

	// Check the data
	assert.Equal(t, activeStakingEvent[0].StakingTxHashHex, results[2].StakingTxHashHex, "expected address to be the same")
	assert.Equal(t, expectedExpireHeight, results[2].ExpireHeight, "expected address to be the same")
}
