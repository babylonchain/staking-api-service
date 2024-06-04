package scripts

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/babylonchain/staking-api-service/internal/config"
	"github.com/babylonchain/staking-api-service/internal/db"
	"github.com/babylonchain/staking-api-service/internal/db/model"
	"github.com/babylonchain/staking-api-service/internal/queue"
	queueClient "github.com/babylonchain/staking-queue-client/client"
	"github.com/rs/zerolog/log"
)

type GenericEvent struct {
	EventType queueClient.EventType `json:"event_type"`
}

func ReplayUnprocessableMessages(ctx context.Context, cfg *config.Config, queues *queue.Queues) {
	dbClient, err := db.New(ctx, cfg.Db)
	if err != nil {
		log.Fatal().Err(err).Msg("error while setting up database client")
	}

	// Fetch unprocessable messages
	cursor, err := dbClient.GetUnprocessableMessages(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to retrieve unprocessable messages")
	}
	defer cursor.Close(ctx)

	// Count the number of unprocessable messages
	messageCount := 0
	for cursor.Next(ctx) {
		messageCount++
	}

	// Inform the user of the number of unprocessable messages
	fmt.Printf("There are %d unprocessable messages.\n", messageCount)
	if messageCount == 0 {
		log.Info().Msg("No unprocessable messages to replay.")
		return
	}

	// Prompt the user for confirmation
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Type 'confirm' to proceed with replaying the messages: ")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input != "confirm" {
		log.Info().Msg("Reprocessing of unprocessable messages aborted.")
		return
	}

	// Reset the cursor to reprocess messages
	cursor, err = dbClient.GetUnprocessableMessages(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to retrieve unprocessable messages")
	}
	defer cursor.Close(ctx)

	// Process each unprocessable message
	for cursor.Next(ctx) {
		var msg model.UnprocessableMessageDocument
		if err := cursor.Decode(&msg); err != nil {
			log.Printf("Failed to decode message: %v", err)
			continue
		}

		var genericEvent GenericEvent
		if err := json.Unmarshal([]byte(msg.MessageBody), &genericEvent); err != nil {
			log.Printf("Failed to unmarshal event message: %v", err)
			continue
		}

		// Process the event message
		if err := processEventMessage(ctx, queues, genericEvent, msg.MessageBody); err != nil {
			log.Printf("Failed to process message: %v", err)
			continue
		}

		// Delete the processed message from the database
		if err := dbClient.DeleteUnprocessableMessage(ctx, msg.Receipt); err != nil {
			log.Printf("Failed to delete unprocessable message: %v", err)
		}
	}

	if err := cursor.Err(); err != nil {
		log.Fatal().Err(err).Msg("Cursor error")
	}

	log.Info().Msg("Reprocessing of unprocessable messages completed.")
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
