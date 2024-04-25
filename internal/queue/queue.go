package queue

import (
	"context"
	"time"

	"github.com/babylonchain/staking-api-service/internal/observability/tracing"
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
	WithdrawStakingQueueClient  client.QueueClient
}

func New(cfg *queueConfig.QueueConfig, service *services.Services) *Queues {
	activeStakingQueueClient, err := client.NewQueueClient(
		cfg, client.ActiveStakingQueueName,
	)
	if err != nil {
		log.Fatal().Err(err).Msg("error while creating ActiveStakingQueueClient")
	}

	expiredStakingQueueClient, err := client.NewQueueClient(
		cfg, client.ExpiredStakingQueueName,
	)
	if err != nil {
		log.Fatal().Err(err).Msg("error while creating ExpiredStakingQueueClient")
	}

	unbondingStakingQueueClient, err := client.NewQueueClient(
		cfg, client.UnbondingStakingQueueName,
	)
	if err != nil {
		log.Fatal().Err(err).Msg("error while creating UnbondingStakingQueueClient")
	}

	withdrawStakingQueueClient, err := client.NewQueueClient(
		cfg, client.WithdrawStakingQueueName,
	)
	if err != nil {
		log.Fatal().Err(err).Msg("error while creating WithdrawStakingQueueClient")
	}

	handlers := handlers.NewQueueHandler(service)
	return &Queues{
		Handlers:                    handlers,
		processingTimeout:           time.Duration(cfg.QueueProcessingTimeout) * time.Second,
		maxRetryAttempts:            cfg.MsgMaxRetryAttempts,
		ActiveStakingQueueClient:    activeStakingQueueClient,
		ExpiredStakingQueueClient:   expiredStakingQueueClient,
		UnbondingStakingQueueClient: unbondingStakingQueueClient,
		WithdrawStakingQueueClient:  withdrawStakingQueueClient,
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
	startQueueMessageProcessing(
		q.WithdrawStakingQueueClient,
		q.Handlers.WithdrawStakingHandler, q.Handlers.HandleUnprocessedMessage,
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
	withdrawnQueueErr := q.WithdrawStakingQueueClient.Stop()
	if withdrawnQueueErr != nil {
		log.Error().Err(withdrawnQueueErr).Str("queueName", q.WithdrawStakingQueueClient.GetQueueName()).Msg("error while stopping queue")
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
			ctx = attachLoggerContext(ctx, message, queueClient)
			// Attach the tracingInfo for the message processing
			_, err := tracing.WrapWithSpan[any](ctx, "message_processing", func() (any, error) {
				return nil, handler(ctx, message.Body)
			})
			if err != nil {
				attempts := message.GetRetryAttempts()
				// We will retry the message if it has not exceeded the max retry attempts
				// otherwise, we will dump the message into db for manual inspection and remove from the queue
				if attempts > maxRetryAttempts {
					log.Ctx(ctx).Error().Err(err).
						Msg("exceeded retry attempts, message will be dumped into db for manual inspection")
					saveUnprocessableMsgErr := unprocessableHandler(ctx, message.Body, message.Receipt)
					if saveUnprocessableMsgErr != nil {
						log.Ctx(ctx).Error().Err(saveUnprocessableMsgErr).
							Msg("error while saving unprocessable message")
						cancel()
						continue
					}
				} else {
					err = queueClient.ReQueueMessage(ctx, message)
					log.Ctx(ctx).Error().Err(err).
						Msg("error while processing message from queue, will be requeued")
					if err != nil {
						log.Ctx(ctx).Error().Err(err).
							Msg("error while requeuing message")
					}
					// TODO: Add metrics for failed message processing
					cancel()
					continue
				}
			}

			delErr := queueClient.DeleteMessage(message.Receipt)
			if delErr != nil {
				// TODO: Add metrics for failed message deletion
				log.Ctx(ctx).Error().Err(delErr).
					Msg("error while deleting message from queue")
			}

			tracingInfo := ctx.Value(tracing.TracingInfoKey)
			logEvent := log.Ctx(ctx).Info()
			if tracingInfo != nil {
				logEvent = logEvent.Interface("tracingInfo", tracingInfo)
			}
			logEvent.Msg("message processed successfully")

			// TODO: Add metrics for successful message processing
			cancel()
		}
		log.Info().Str("queueName", queueClient.GetQueueName()).Msg("stopped receiving messages from queue")
	}()
}

func attachLoggerContext(ctx context.Context, message client.QueueMessage, queueClient client.QueueClient) context.Context {
	ctx = tracing.AttachTracingIntoContext(ctx)

	traceId := ctx.Value(tracing.TraceIdKey)
	return log.With().
		Str("receipt", message.Receipt).
		Str("queueName", queueClient.GetQueueName()).
		Interface("traceId", traceId).
		Logger().WithContext(ctx)
}
