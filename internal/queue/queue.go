package queue

import (
	"context"
	"time"

	"github.com/babylonchain/staking-api-service/internal/config"
	"github.com/babylonchain/staking-api-service/internal/queue/handlers"
	"github.com/babylonchain/staking-api-service/internal/services"
	"github.com/babylonchain/staking-queue-client/client"
	"github.com/rs/zerolog/log"
)

type MessageHandler func(ctx context.Context, messageBody string) error

type Queues struct {
	Handlers                  *handlers.QueueHandler
	processingTimeout         time.Duration
	maxRetryAttempts          int32
	ActiveStakingQueueClient  client.QueueClient
	ExpiredStakingQueueClient client.QueueClient
}

func New(cfg config.QueueConfig, service *services.Services) *Queues {
	activeStakingQueueClient, err := client.NewQueueClient(
		cfg.Url, cfg.QueueUser, cfg.QueuePassword, client.ActiveStakingQueueName,
	)
	if err != nil {
		log.Fatal().Err(err).Msg("error while creating ActiveStakingQueueClient")
	}

	expiredStakingQueueClient, err := client.NewQueueClient(
		cfg.Url, cfg.QueueUser, cfg.QueuePassword, client.ExpiredStakingQueueName,
	)
	if err != nil {
		log.Fatal().Err(err).Msg("error while creating ExpiredStakingQueueClient")
	}

	handlers := handlers.NewQueueHandler(service)
	return &Queues{
		Handlers:                  handlers,
		processingTimeout:         time.Duration(cfg.QueueProcessingTimeout) * time.Second,
		maxRetryAttempts:          cfg.MaxRetryAttempts,
		ActiveStakingQueueClient:  activeStakingQueueClient,
		ExpiredStakingQueueClient: expiredStakingQueueClient,
	}
}

// Start all message processing
func (q *Queues) StartReceivingMessages() {
	// start processing messages from the active staking queue
	startQueueMessageProcessing(q.ActiveStakingQueueClient, q.Handlers.ActiveStakingHandler, q)
	startQueueMessageProcessing(q.ExpiredStakingQueueClient, q.Handlers.ExpiredStakingHandler, q)
	// ...add more queues here
}

// Turn off all message processing
func (q *Queues) StopReceivingMessages() {
	activeQueueErr := q.ActiveStakingQueueClient.Stop()
	if activeQueueErr != nil {
		log.Error().Err(activeQueueErr).Str("queueName", q.ActiveStakingQueueClient.GetQueueName()).Msg("error while stopping queue")
	}
	expiredQueueErr := q.ExpiredStakingQueueClient.Stop()
	if expiredQueueErr != nil {
		log.Error().Err(expiredQueueErr).Str("queueName", q.ExpiredStakingQueueClient.GetQueueName()).Msg("error while stopping queue")
	}
	// ...add more queues here
}

func startQueueMessageProcessing(
	queueClient client.QueueClient, handler MessageHandler, q *Queues) {
	messagesChan, err := queueClient.ReceiveMessages()
	log.Info().Str("queueName", queueClient.GetQueueName()).Msg("start receiving messages from queue")
	if err != nil {
		log.Fatal().Err(err).Str("queueName", queueClient.GetQueueName()).Msg("error setting up message channel from queue")
	}

	go func() {
		for message := range messagesChan {
			// For each message, create a new context with a deadline or timeout
			ctx, cancel := context.WithTimeout(context.Background(), q.processingTimeout)
			err := handler(ctx, message.Body)
			if err != nil {
				attempts := message.GetRetryAttempts()
				// We will retry the message if it has not exceeded the max retry attempts
				// otherwise, we will dump the message into db for manual inspection and remove from the queue
				if attempts > q.maxRetryAttempts {
					log.Error().Err(err).Str("queueName", queueClient.GetQueueName()).
						Str("receipt", message.Receipt).Msg("exceeded retry attempts, message will be dumped into db for manual inspection")
					saveUnprocessableMsgErr := q.Handlers.HandleUnprocessedMessage(ctx, message.Body, message.Receipt)
					if saveUnprocessableMsgErr != nil {
						log.Error().Err(saveUnprocessableMsgErr).Str("queueName", queueClient.GetQueueName()).
							Str("receipt", message.Receipt).Msg("error while saving unprocessable message")
						cancel()
						continue
					}
				} else {
					// TODO: Below requeue is a workaround
					// it need to be handled by https://github.com/babylonchain/staking-api-service/issues/38
					time.Sleep(5 * time.Second)
					err = queueClient.ReQueueMessage(ctx, message)
					log.Error().Err(err).Str("queueName", queueClient.GetQueueName()).
						Str("receipt", message.Receipt).
						Msg("error while processing message from queue, will be requeued")
					if err != nil {
						log.Error().Err(err).Str("queueName", queueClient.GetQueueName()).
							Str("receipt", message.Receipt).Msg("error while requeuing message")
					}
					// TODO: Add metrics for failed message processing
					cancel()
					continue
				}
			}

			delErr := queueClient.DeleteMessage(message.Receipt)
			if delErr != nil {
				// TODO: Add metrics for failed message deletion
				log.Error().Err(delErr).Str("queueName", queueClient.GetQueueName()).
					Str("receipt", message.Receipt).Msg("error while deleting message from queue")
			}

			// TODO: Add metrics for successful message processing
			cancel()
		}
		log.Info().Str("queueName", queueClient.GetQueueName()).Msg("stopped receiving messages from queue")
	}()
}
