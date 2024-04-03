package handlers

import (
	"context"
	"encoding/json"

	"github.com/rs/zerolog/log"

	queueClient "github.com/babylonchain/staking-api-service/internal/queue/client"
)

func (h *QueueHandler) ActiveStakingHandler(ctx context.Context, messageBody string) error {
	// Parse the message body into ActiveStakingEvent
	var activeStakingEvent queueClient.ActiveStakingEvent
	err := json.Unmarshal([]byte(messageBody), &activeStakingEvent)
	if err != nil {
		log.Error().Err(err).Msg("Failed to unmarshal the message body into ActiveStakingEvent")
		return err
	}

	err = h.Services.SaveActiveStakingDelegation(
		ctx, activeStakingEvent.StakingTxHashHex, activeStakingEvent.StakerPkHex,
		activeStakingEvent.FinalityProviderPkHex, activeStakingEvent.StakingValue,
		activeStakingEvent.StakingStartHeight, activeStakingEvent.StakingTimeLock,
		activeStakingEvent.StakingOutputIndex,
	)
	if err != nil {
		return err
	}

	err = h.Services.ProcessStakingStatsCalculation(ctx, activeStakingEvent)
	if err != nil {
		log.Error().Err(err).Msg("Failed to process staking stats calculation")
		return err
	}

	err = h.Services.ProcessExpireCheck(
		ctx, activeStakingEvent.StakingTxHashHex,
		activeStakingEvent.StakingStartHeight,
		activeStakingEvent.StakingTimeLock,
		queueClient.ActiveTxType.ToString(),
	)
	if err != nil {
		log.Error().Err(err).Msg("Failed to process expire check")
		return err
	}

	return nil
}
