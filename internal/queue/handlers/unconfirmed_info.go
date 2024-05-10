package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/babylonchain/staking-api-service/internal/types"
	queueClient "github.com/babylonchain/staking-queue-client/client"
	"github.com/rs/zerolog/log"
)

func (h *QueueHandler) UnconfirmedInfoHandler(ctx context.Context, messageBody string) *types.Error {
	var unconfirmedInfo queueClient.UnconfirmedInfoEvent
	err := json.Unmarshal([]byte(messageBody), &unconfirmedInfo)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("Failed to unmarshal the message body into unconfirmedInfo")
		return types.NewError(http.StatusBadRequest, types.BadRequest, err)
	}
	statsErr := h.Services.ProcessUnconfirmedTvlStats(ctx, unconfirmedInfo.Height, unconfirmedInfo.ActiveTvl)
	if statsErr != nil {
		log.Error().Err(statsErr).Msg("Failed to process unconfirmed tvl stats")
		return types.NewInternalServiceError(statsErr)
	}
	return nil
}
