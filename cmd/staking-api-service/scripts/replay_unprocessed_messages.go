package scripts

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/babylonchain/staking-api-service/internal/config"
	"github.com/babylonchain/staking-api-service/internal/db"
	"github.com/babylonchain/staking-api-service/internal/queue"
	queueClient "github.com/babylonchain/staking-queue-client/client"
	"github.com/rs/zerolog/log"
)

type GenericEvent struct {
	EventType queueClient.EventType `json:"event_type"`
}

func ReplayUnprocessableMessages(ctx context.Context, cfg *config.Config, queues *queue.Queues, dbClient db.DBClient) (err error) {
	// Fetch unprocessable messages
	unprocessableMessages, err := dbClient.FindUnprocessableMessages(ctx)
	if err != nil {
		return errors.New("failed to retrieve unprocessable messages")
	}

	// Get the message count
	messageCount := len(unprocessableMessages)

	// Inform the user of the number of unprocessable messages
	fmt.Printf("There are %d unprocessable messages.\n", messageCount)
	if messageCount == 0 {
		return errors.New("no unprocessable messages to replay")
	}

	// Process each unprocessable message
	for _, msg := range unprocessableMessages {
		var genericEvent GenericEvent
		if err := json.Unmarshal([]byte(msg.MessageBody), &genericEvent); err != nil {
			log.Printf("Failed to unmarshal event message: %v", err)
			return errors.New("failed to unmarshal event message")
		}

		// Process the event message
		if err := processEventMessage(ctx, queues, genericEvent, msg.MessageBody); err != nil {
			log.Printf("Failed to process message: %v", err)
			return errors.New("failed to process message")
		}

		// Delete the processed message from the database
		if err := dbClient.DeleteUnprocessableMessage(ctx, msg.Receipt); err != nil {	
			log.Printf("Failed to delete unprocessable message: %v", err)
			return errors.New("failed to delete unprocessable message")
		}
	}

	log.Info().Msg("Reprocessing of unprocessable messages completed.")	
	return
}

// processEventMessage processes the event message based on its EventType.
func processEventMessage(ctx context.Context, queues *queue.Queues, event GenericEvent, messageBody string) error {
	// Define a map of event types to their corresponding SendMessage functions.
	eventHandlers := map[queueClient.EventType]func(context.Context, string) error{
		queueClient.ActiveStakingEventType:    queues.ActiveStakingQueueClient.SendMessage,
		queueClient.UnbondingStakingEventType: queues.UnbondingStakingQueueClient.SendMessage,
		queueClient.WithdrawStakingEventType:  queues.WithdrawStakingQueueClient.SendMessage,
		queueClient.ExpiredStakingEventType:   queues.ExpiredStakingQueueClient.SendMessage,
		queueClient.StatsEventType:            queues.StatsQueueClient.SendMessage,
		queueClient.BtcInfoEventType:          queues.BtcInfoQueueClient.SendMessage,
	}

	// Get the appropriate handler based on the event type.
	handler, ok := eventHandlers[event.EventType]
	if !ok {
		return fmt.Errorf("unknown event type: %v", event.EventType)
	}

	// Call the handler with the context and message body.
	return handler(ctx, messageBody)
}
