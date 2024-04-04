package handlers

import (
	"context"

	"github.com/babylonchain/staking-api-service/internal/services"
)

type QueueHandler struct {
	Services *services.Services
}

type MessageHandler func(ctx context.Context, messageBody string) error
type UnprocessableMessageHandler func(ctx context.Context, messageBody, receipt string) error

func NewQueueHandler(services *services.Services) *QueueHandler {
	return &QueueHandler{
		Services: services,
	}
}

func (qh *QueueHandler) HandleUnprocessedMessage(ctx context.Context, messageBody, receipt string) error {
	return qh.Services.SaveUnprocessableMessages(ctx, messageBody, receipt)
}
