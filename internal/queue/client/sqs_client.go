package client

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
)

type SQSClient struct {
	client   *sqs.SQS
	queueURL string
	// isRunning bool
}

func NewSQSClient(queueURL string, region string) *SQSClient {
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(region),
	}))
	client := sqs.New(sess)

	return &SQSClient{
		client:   client,
		queueURL: queueURL,
	}
}

func (c *SQSClient) ReceiveMessages() ([]QueueMessage, error) {
	output, err := c.client.ReceiveMessage(&sqs.ReceiveMessageInput{
		QueueUrl:        &c.queueURL,
		WaitTimeSeconds: aws.Int64(20), // TODO: move to config
	})
	if err != nil {
		return nil, err
	}

	// Build the []QueueMessage based on the output.
	var messages []QueueMessage
	for _, message := range output.Messages {
		messages = append(messages, QueueMessage{
			Body:    *message.Body,
			Receipt: *message.ReceiptHandle,
		})
	}
	return messages, nil
}

func (c *SQSClient) DeleteMessage(receipt string) error {
	_, err := c.client.DeleteMessage(&sqs.DeleteMessageInput{
		QueueUrl:      &c.queueURL,
		ReceiptHandle: &receipt,
	})
	return err
}

func (c *SQSClient) SendMessage(messageBody string) error {
	_, err := c.client.SendMessage(&sqs.SendMessageInput{
		QueueUrl:    &c.queueURL,
		MessageBody: aws.String(messageBody),
	})
	return err
}

// func (c *SQSClient) StartReceivingMessages(handler handlers.MessageHandler) error {
// 	c.isRunning = true
// 	go func() {
// 		for c.isRunning {
// 			output, err := c.client.ReceiveMessage(&sqs.ReceiveMessageInput{
// 				QueueUrl:        &c.queueURL,
// 				WaitTimeSeconds: aws.Int64(20), // TODO: move to config
// 			})
// 			if err != nil || len(output.Messages) == 0 {
// 				// No messages received, hence continue the next iteration
// 				continue
// 			}

// 			// By default, we will only have a single message in the output.
// 			// However, we are iterating over the messages to handle the case
// 			for _, message := range output.Messages {
// 				err = handler(*message.Body)
// 				if err != nil {
// 					// TODO: We log the error and emit metrics
// 					continue
// 				}

// 				// Delete message after processing
// 				_, delErr := c.client.DeleteMessage(&sqs.DeleteMessageInput{
// 					QueueUrl:      &c.queueURL,
// 					ReceiptHandle: message.ReceiptHandle,
// 				})
// 				if delErr != nil {
// 					// Handle error
// 				}
// 			}
// 		}
// 	}()
// 	return nil
// }

// func (c *SQSClient) StopReceivingMessages() error {
// 	c.isRunning = false
// 	return nil
// }
