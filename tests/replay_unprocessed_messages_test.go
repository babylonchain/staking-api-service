package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/babylonchain/staking-api-service/cmd/staking-api-service/scripts"
	testmock "github.com/babylonchain/staking-api-service/tests/mocks"
	"github.com/babylonchain/staking-queue-client/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestReplayUnprocessableMessages(t *testing.T) {    
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	mockDB := new(testmock.DBClient)
	mockDB.On("FindUnprocessableMessages", mock.Anything).Return(nil, errors.New("just an error"))

	testServer := setupTestServer(t, &TestServerDependency{MockDbClient: mockDB})
	defer testServer.Close()

	activeStakingEvent := getTestActiveStakingEvent()

	// Send a test message to the queue
	sendTestMessage(testServer.Queues.ActiveStakingQueueClient, []client.ActiveStakingEvent{*activeStakingEvent})
	err := addQueueJobToUnprocessedMessage(ctx, testServer.Queues.ActiveStakingQueueClient, testServer)
	assert.NoError(t, err, "addQueueJobToUnprocessedMessage should not return an error")

	time.Sleep(2 * time.Second)

	// Capture log output to verify correct log messages
	err = scripts.ReplayUnprocessableMessages(ctx, testServer.Config, testServer.Queues, mockDB)

	// Assert that the function returned an error, indicating it logged the error and did not proceed further
	assert.Error(t, err, "ReplayUnprocessableMessages should return an error if FindUnprocessableMessages fails")
	assert.Equal(t, "failed to retrieve unprocessable messages", err.Error())

	mockDB.AssertCalled(t, "FindUnprocessableMessages", mock.Anything)
	mockDB.AssertNotCalled(t, "DeleteUnprocessableMessage", mock.Anything, mock.Anything)
}

// FailJob simulates the failure of a specific job message
func addQueueJobToUnprocessedMessage(ctx context.Context, queueClient client.QueueClient, testServer *TestServer) error {
	// Simulate job failure
	messagesChan, err := queueClient.ReceiveMessages()
	if err != nil {
			return err
	}

	for message := range messagesChan {
			// Save job to unprocessable messages table
			if err := testServer.Queues.Handlers.HandleUnprocessedMessage(ctx, message.Body, message.Receipt); err != nil {
					return err
			}
			if delErr := queueClient.DeleteMessage(message.Receipt); delErr != nil {
					return delErr
			}
	}

	return nil
}