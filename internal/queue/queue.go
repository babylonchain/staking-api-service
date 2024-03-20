package queue

import (
	"context"
	"time"

	"github.com/babylonchain/staking-api-service/internal/config"
	"github.com/babylonchain/staking-api-service/internal/queue/client"
	"github.com/babylonchain/staking-api-service/internal/queue/handlers"
	"github.com/babylonchain/staking-api-service/internal/services"
	"github.com/rs/zerolog/log"
)

type MessageHandler func(ctx context.Context, messageBody string) error

type Queues struct {
	ActiveStakingQueueClient client.QueueClient
	Handlers                 *handlers.QueueHandler
	processingTimeout        time.Duration
}

func New(cfg config.QueueConfig, service *services.Services) *Queues {
	activeStakingQueueClient, err := client.NewQueueClient(
		cfg.Url, cfg.QueueUser, cfg.QueuePassword, cfg.ActiveStakingQueueName,
	)
	if err != nil {
		log.Fatal().Err(err).Msg("error while creating ActiveStakingQueueClient")
	}
	handlers := handlers.NewQueueHandler(service)
	return &Queues{
		ActiveStakingQueueClient: activeStakingQueueClient,
		Handlers:                 handlers,
		processingTimeout:        time.Duration(cfg.QueueProcessingTimeout) * time.Second,
	}
}

// Start all message processing
func (q *Queues) StartReceivingMessages() {
	// start processing messages from the active staking queue
	startQueueMessageProcessing(q.ActiveStakingQueueClient, q.Handlers.ActiveStakingHandler, q.processingTimeout)
	// ...add more queues here
}

// Turn off all message processing
func (q *Queues) StopReceivingMessages() {
	q.ActiveStakingQueueClient.Stop()
}

func startQueueMessageProcessing(
	queueClient client.QueueClient, handler MessageHandler, timeout time.Duration) {
	messagesChan, err := queueClient.ReceiveMessages()
	log.Info().Str("queueName", queueClient.GetQueueName()).Msg("start receiving messages from queue")
	if err != nil {
		log.Fatal().Err(err).Str("queueName", queueClient.GetQueueName()).Msg("error setting up message channel from queue")
	}

	go func() {
		for message := range messagesChan {
			// For each message, create a new context with a deadline or timeout
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			err := handler(ctx, message.Body)
			if err != nil {
				log.Error().Err(err).Str("queueName", queueClient.GetQueueName()).Msg("error while processing message from queue")
				cancel()
				continue
			}

			delErr := queueClient.DeleteMessage(message.Receipt)
			if delErr != nil {
				log.Error().Err(delErr).Str("queueName", queueClient.GetQueueName()).Msg("error while deleting message from queue")
			}
			cancel()
		}
	}()
}
