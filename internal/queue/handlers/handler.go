package handlers

import (
	"context"
	"encoding/json"

	"github.com/babylonchain/staking-api-service/internal/services"
	"github.com/babylonchain/staking-queue-client/client"
	"github.com/rs/zerolog/log"
)

type QueueHandler struct {
	Services       *services.Services
	emitStatsEvent func(ctx context.Context, messageBody string) error
}

type MessageHandler func(ctx context.Context, messageBody string) error
type UnprocessableMessageHandler func(ctx context.Context, messageBody, receipt string) error

func NewQueueHandler(
	services *services.Services,
	emitStats func(ctx context.Context, messageBody string) error,
) *QueueHandler {
	return &QueueHandler{
		Services:       services,
		emitStatsEvent: emitStats,
	}
}

func (qh *QueueHandler) HandleUnprocessedMessage(ctx context.Context, messageBody, receipt string) error {
	return qh.Services.SaveUnprocessableMessages(ctx, messageBody, receipt)
}

func (qh *QueueHandler) EmitStatsEvent(ctx context.Context, statsEvent client.StatsEvent) error {
	jsonData, err := json.Marshal(statsEvent)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("Failed to marshal the stats event")
		return err
	}
	return qh.emitStatsEvent(ctx, string(jsonData))
}
