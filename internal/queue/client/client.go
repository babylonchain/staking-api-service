package client

import "context"

type QueueMessage struct {
	Body    string
	Receipt string
}

// A common interface for queue clients regardless if it's a SQS, RabbitMQ, etc.
type QueueClient interface {
	SendMessage(ctx context.Context, messageBody string) error
	ReceiveMessages() (<-chan QueueMessage, error)
	DeleteMessage(receipt string) error
	Stop()
	GetQueueName() string
}

func NewQueueClient(queueURL, user, pass, queueName string) (QueueClient, error) {
	return NewRabbitMqClient(queueURL, user, pass, queueName)
}
