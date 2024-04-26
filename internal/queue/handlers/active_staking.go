package handlers

import (
	"context"
	"encoding/json"

	"github.com/babylonchain/staking-api-service/internal/types"
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
		log.Ctx(ctx).Error().Err(err).Msg("Failed to unmarshal the message body into ActiveStakingEvent")
		return err
	}

	// Check if delegation already exists
	exist, delError := h.Services.IsDelegationPresent(ctx, activeStakingEvent.StakingTxHashHex)
	if delError != nil {
		return delError
	}
	if exist {
		// Ignore the message as the delegation already exists. This is a duplicate message
		return nil
	}

	// Perform the async stats calculation by emit the stats event
	statsError := h.EmitStatsEvent(ctx, queueClient.NewStatsEvent(
		activeStakingEvent.StakingTxHashHex,
		activeStakingEvent.StakerPkHex,
		activeStakingEvent.FinalityProviderPkHex,
		activeStakingEvent.StakingValue,
		types.Active.ToString(),
	))
	if statsError != nil {
		log.Ctx(ctx).Error().Err(statsError).Msg("Failed to emit stats event for active staking")
		return statsError
	}

	// Perform the async timelock expire check
	expireCheckError := h.Services.ProcessExpireCheck(
		ctx, activeStakingEvent.StakingTxHashHex,
		activeStakingEvent.StakingStartHeight,
		activeStakingEvent.StakingTimeLock,
		types.ActiveTxType,
	)
	if expireCheckError != nil {
		log.Ctx(ctx).Error().Err(expireCheckError).Msg("Failed to process expire check")
		return expireCheckError
	}

	// Save the active staking delegation. This is the final step in the active staking event processing
	// Please refer to the README.md for the details on the active staking event processing workflow
	err = h.Services.SaveActiveStakingDelegation(
		ctx, activeStakingEvent.StakingTxHashHex, activeStakingEvent.StakerPkHex,
		activeStakingEvent.FinalityProviderPkHex, activeStakingEvent.StakingValue,
		activeStakingEvent.StakingStartHeight, activeStakingEvent.StakingStartTimestamp,
		activeStakingEvent.StakingTimeLock, activeStakingEvent.StakingOutputIndex,
		activeStakingEvent.StakingTxHex,
	)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("Failed to save active staking delegation")
		return err
	}

	return nil
}
