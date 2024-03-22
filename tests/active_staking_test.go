package tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/babylonchain/staking-api-service/internal/queue/client"
)

func TestActiveStakingHandler(t *testing.T) {
	activeStakingEvent := buildActiveStakingEvent(1)
	server, queues := setupTestServer(t, nil)
	defer server.Close()
	sendTestMessage(queues.ActiveStakingQueueClient, activeStakingEvent)

	// Wait for 10 seconds to make sure the message is processed
	time.Sleep(10 * time.Second)
	// Test the API

}

func buildActiveStakingEvent(numOfEvenet int) []client.ActiveStakingEvent {
	var activeStakingEvents []client.ActiveStakingEvent
	for i := 0; i < numOfEvenet; i++ {
		activeStakingEvent := client.ActiveStakingEvent{
			EventType:             client.ActiveStakingEventType,
			StakingTxHex:          "0x1234567890abcdef" + fmt.Sprint(i),
			StakerPkHex:           "0xabcdef1234567890" + fmt.Sprint(i),
			FinalityProviderPkHex: "0xabcdef1234567890" + fmt.Sprint(i),
			StakingValue:          1 + uint64(i),
			StakingStartkHeight:   100 + uint64(i),
			StakingTimeLock:       200 + uint16(i),
		}
		activeStakingEvents = append(activeStakingEvents, activeStakingEvent)
	}
	return activeStakingEvents
}
