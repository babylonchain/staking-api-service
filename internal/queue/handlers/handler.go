package handlers

import "github.com/babylonchain/staking-api-service/internal/services"

type QueueHandler struct {
	Services *services.Services
}

type MessageHandler func(messageBody string) error

func NewQueueHandler(services *services.Services) *QueueHandler {
	return &QueueHandler{
		Services: services,
	}
}
