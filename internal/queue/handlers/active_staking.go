package handlers

import (
	"context"
	"encoding/json"

	queueClient "github.com/babylonchain/staking-api-service/internal/queue/client"
	"github.com/rs/zerolog/log"
)

func (h *QueueHandler) ActiveStakingHandler(ctx context.Context, messageBody string) error {
	// Parse the message body into ActiveStakingEvent
	var activeStakingEvent queueClient.ActiveStakingEvent
	err := json.Unmarshal([]byte(messageBody), &activeStakingEvent)
	if err != nil {
		log.Error().Err(err).Msg("Failed to unmarshal the message body into ActiveStakingEvent")
		return err
	}

	err = h.Services.SaveActiveStakingDelegation(ctx, activeStakingEvent)
	if err != nil {
		return err
	}

	err = h.Services.ProcessStakingStatsCalculation(ctx, activeStakingEvent)
	if err != nil {
		return err
	}

	err = h.Services.ProcessExpireCheck(
		ctx, activeStakingEvent.StakingTxHex,
		activeStakingEvent.StakingStartkHeight, activeStakingEvent.StakingTimeLock,
	)
	if err != nil {
		return err
	}

	return nil
}
