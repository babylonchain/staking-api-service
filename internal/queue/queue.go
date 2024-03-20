package queue

import (
	"github.com/babylonchain/staking-api-service/internal/config"
	"github.com/babylonchain/staking-api-service/internal/db"
	"github.com/babylonchain/staking-api-service/internal/queue/client"
	"github.com/babylonchain/staking-api-service/internal/queue/handlers"
	"github.com/rs/zerolog/log"
)

type MessageHandler func(messageBody string) error

type Queues struct {
	ActiveStakingQueueClient client.QueueClient
	Handlers                 *handlers.QueueHandler
	isRunning                bool
}

func New(cfg *config.QueueConfig, dbClient *db.DBClient) *Queues {
	activeStakingQueueClient := client.NewQueueClient(cfg.ActiveStakingQueueUrl, cfg.Region)
	handlers := handlers.NewQueueHandler(dbClient)
	return &Queues{
		ActiveStakingQueueClient: activeStakingQueueClient,
		Handlers:                 handlers,
	}
}

// Start all message processing
func (q *Queues) StartReceivingMessages() {
	q.isRunning = true
	q.processingActiveStakingTransactions()
	// TODO: Add more queues here
}

// Turn off all message processing
func (q *Queues) StopReceivingMessages() {
	q.isRunning = false
}

func (q *Queues) processingActiveStakingTransactions() {
	go func() {
		for q.isRunning {
			// TODO: Manually create a ctx so that we can link all the spans to the same trace in the log
			messages, err := q.ActiveStakingQueueClient.ReceiveMessages()
			if err != nil {
				log.Err(err).Msg("error while receiving messages from ActiveStakingQueue")
				continue
			}
			if len(messages) == 0 {
				// No messages received, hence continue the next iteration
				continue
			}

			// By default, we will only have a single message in the output.
			// However, we are iterating over the messages to handle the case
			for _, message := range messages {
				err = q.Handlers.ActiveStakingHandler(message.Body)
				if err != nil {
					log.Error().Err(err).Msg("error while processing message from ActiveStakingQueue")
					continue
				}

				delErr := q.ActiveStakingQueueClient.DeleteMessage(message.Receipt)
				if delErr != nil {
					log.Error().Err(delErr).Msg("error while deleting message from ActiveStakingQueue")
				}
			}
		}
	}()
}
