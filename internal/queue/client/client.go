package client

type QueueMessage struct {
	Body    string
	Receipt string
}

// A common interface for queue clients regardless if it's a SQS, RabbitMQ, etc.
type QueueClient interface {
	SendMessage(messageBody string) error
	ReceiveMessages() ([]QueueMessage, error)
	DeleteMessage(receipt string) error
}

func NewQueueClient(queueURL string, region string) QueueClient {
	return NewSQSClient(queueURL, region)
}
