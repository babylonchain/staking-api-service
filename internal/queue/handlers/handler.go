package handlers

import (
	"github.com/babylonchain/staking-api-service/internal/db"
)

type QueueHandler struct {
	DBClient *db.DBClient
}

type MessageHandler func(messageBody string) error

func NewQueueHandler(dbClient *db.DBClient) *QueueHandler {
	return &QueueHandler{
		DBClient: dbClient,
	}
}
