package queue

import (
	"context"
	"time"

	"github.com/babylonchain/staking-api-service/internal/queue/handlers"
	"github.com/babylonchain/staking-api-service/internal/services"
	"github.com/babylonchain/staking-queue-client/client"
	queueConfig "github.com/babylonchain/staking-queue-client/config"
	"github.com/rs/zerolog/log"
)

type Queues struct {
	Handlers                    *handlers.QueueHandler
	processingTimeout           time.Duration
	maxRetryAttempts            int32
	ActiveStakingQueueClient    client.QueueClient
	ExpiredStakingQueueClient   client.QueueClient
	UnbondingStakingQueueClient client.QueueClient
}

func New(cfg queueConfig.QueueConfig, service *services.Services) *Queues {
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

	unbondingStakingQueueClient, err := client.NewQueueClient(
		cfg.Url, cfg.QueueUser, cfg.QueuePassword, client.UnbondingStakingQueueName,
	)
	if err != nil {
		log.Fatal().Err(err).Msg("error while creating UnbondingStakingQueueClient")
	}

	handlers := handlers.NewQueueHandler(service)
	return &Queues{
		Handlers:                    handlers,
		processingTimeout:           time.Duration(cfg.QueueProcessingTimeout) * time.Second,
		maxRetryAttempts:            cfg.MsgMaxRetryAttempts,
		ActiveStakingQueueClient:    activeStakingQueueClient,
		ExpiredStakingQueueClient:   expiredStakingQueueClient,
		UnbondingStakingQueueClient: unbondingStakingQueueClient,
	}
}

// Start all message processing
func (q *Queues) StartReceivingMessages() {
	// start processing messages from the active staking queue
	startQueueMessageProcessing(
		q.ActiveStakingQueueClient,
		q.Handlers.ActiveStakingHandler, q.Handlers.HandleUnprocessedMessage,
		q.maxRetryAttempts, q.processingTimeout,
	)
	startQueueMessageProcessing(
		q.ExpiredStakingQueueClient,
		q.Handlers.ExpiredStakingHandler, q.Handlers.HandleUnprocessedMessage,
		q.maxRetryAttempts, q.processingTimeout,
	)
	startQueueMessageProcessing(
		q.UnbondingStakingQueueClient,
		q.Handlers.UnbondingStakingHandler, q.Handlers.HandleUnprocessedMessage,
		q.maxRetryAttempts, q.processingTimeout,
	)
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
	unbondingQueueErr := q.UnbondingStakingQueueClient.Stop()
	if unbondingQueueErr != nil {
		log.Error().Err(unbondingQueueErr).Str("queueName", q.UnbondingStakingQueueClient.GetQueueName()).Msg("error while stopping queue")
	}
	// ...add more queues here
}

func startQueueMessageProcessing(
	queueClient client.QueueClient,
	handler handlers.MessageHandler, unprocessableHandler handlers.UnprocessableMessageHandler,
	maxRetryAttempts int32, processingTimeout time.Duration,
) {
	messagesChan, err := queueClient.ReceiveMessages()
	log.Info().Str("queueName", queueClient.GetQueueName()).Msg("start receiving messages from queue")
	if err != nil {
		log.Fatal().Err(err).Str("queueName", queueClient.GetQueueName()).Msg("error setting up message channel from queue")
	}

	go func() {
		for message := range messagesChan {
			// For each message, create a new context with a deadline or timeout
			ctx, cancel := context.WithTimeout(context.Background(), processingTimeout)
			err := handler(ctx, message.Body)
			if err != nil {
				attempts := message.GetRetryAttempts()
				// We will retry the message if it has not exceeded the max retry attempts
				// otherwise, we will dump the message into db for manual inspection and remove from the queue
				if attempts > maxRetryAttempts {
					log.Ctx(ctx).Error().Err(err).Str("queueName", queueClient.GetQueueName()).
						Str("receipt", message.Receipt).Msg("exceeded retry attempts, message will be dumped into db for manual inspection")
					saveUnprocessableMsgErr := unprocessableHandler(ctx, message.Body, message.Receipt)
					if saveUnprocessableMsgErr != nil {
						log.Ctx(ctx).Error().Err(saveUnprocessableMsgErr).Str("queueName", queueClient.GetQueueName()).
							Str("receipt", message.Receipt).Msg("error while saving unprocessable message")
						cancel()
						continue
					}
				} else {
					// TODO: Below requeue is a workaround
					// it need to be handled by https://github.com/babylonchain/staking-api-service/issues/38
					time.Sleep(5 * time.Second)
					err = queueClient.ReQueueMessage(ctx, message)
					log.Ctx(ctx).Error().Err(err).Str("queueName", queueClient.GetQueueName()).
						Str("receipt", message.Receipt).
						Msg("error while processing message from queue, will be requeued")
					if err != nil {
						log.Ctx(ctx).Error().Err(err).Str("queueName", queueClient.GetQueueName()).
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
				log.Ctx(ctx).Error().Err(delErr).Str("queueName", queueClient.GetQueueName()).
					Str("receipt", message.Receipt).Msg("error while deleting message from queue")
			}

			// TODO: Add metrics for successful message processing
			cancel()
		}
		log.Info().Str("queueName", queueClient.GetQueueName()).Msg("stopped receiving messages from queue")
	}()
}
