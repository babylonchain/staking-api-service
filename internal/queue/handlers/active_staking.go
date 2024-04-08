package handlers

import (
	"context"
	"encoding/json"

	"github.com/babylonchain/staking-api-service/internal/utils"
	queueClient "github.com/babylonchain/staking-queue-client/client"
	"github.com/rs/zerolog/log"
)

// ActiveStakingHandler handles the active staking event
// This handler is designed to be idempotent, capable of handling duplicate messages gracefully.
// It can also resume from the next step if a previous step fails, ensuring robustness in the event processing workflow.
func (h *QueueHandler) ActiveStakingHandler(ctx context.Context, messageBody string) error {
	// Parse the message body into ActiveStakingEvent
	var activeStakingEvent queueClient.ActiveStakingEvent
	err := json.Unmarshal([]byte(messageBody), &activeStakingEvent)
	if err != nil {
		log.Error().Err(err).Msg("Failed to unmarshal the message body into ActiveStakingEvent")
		return err
	}

	// Check if delegation is already exist
	exist, delErro := h.Services.IsDelegationPresent(ctx, activeStakingEvent.StakingTxHashHex)
	if delErro != nil {
		return delErro
	}
	if exist {
		// Ignore the message as the delegation already exists. This is a duplicate message
		return nil
	}

	// TODO: To be replaced with the epoch timestamp
	activeStakingTimestamp, err := utils.ParseTimestampToIsoFormat(activeStakingEvent.StakingStartTimestamp)
	if err != nil {
		log.Error().Err(err).Msg("Failed to parse the active staking timestamp into ISO8601 format")
		return err
	}

	// Perform the async stats calculation
	err = h.Services.ProcessStakingStatsCalculation(ctx, activeStakingEvent)
	if err != nil {
		log.Error().Err(err).Msg("Failed to process staking stats calculation")
		return err
	}

	// Perform the async timelock expire check
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

	err = h.Services.SaveActiveStakingDelegation(
		ctx, activeStakingEvent.StakingTxHashHex, activeStakingEvent.StakerPkHex,
		activeStakingEvent.FinalityProviderPkHex, activeStakingEvent.StakingValue,
		activeStakingEvent.StakingStartHeight, activeStakingTimestamp,
		activeStakingEvent.StakingTimeLock, activeStakingEvent.StakingOutputIndex,
		activeStakingEvent.StakingTxHex,
	)
	if err != nil {
		return err
	}

	return nil
}
