package handlers

import (
	"context"
	"encoding/json"

	"github.com/babylonchain/staking-api-service/internal/types"
	"github.com/babylonchain/staking-api-service/internal/utils"
	queueClient "github.com/babylonchain/staking-queue-client/client"
	"github.com/rs/zerolog/log"
)

func (h *QueueHandler) UnbondingStakingHandler(ctx context.Context, messageBody string) error {
	var unbondingStakingEvent queueClient.UnbondingStakingEvent
	err := json.Unmarshal([]byte(messageBody), &unbondingStakingEvent)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("Failed to unmarshal the message body into unbondingStakingEvent")
		return err
	}

	// Check if the delegation is in the right state to process the unbonding event
	state, stateErr := h.Services.GetDelegationState(ctx, unbondingStakingEvent.StakingTxHashHex)
	// Requeue if found any error. Including not found error
	if stateErr != nil {
		return stateErr
	}
	if utils.Contains[types.DelegationState](utils.OutdatedStatesForUnbonding(), state) {
		// Ignore the message as the delegation state already passed the unbonding state. This is an outdated duplication
		return nil
	}

	err = h.Services.ProcessExpireCheck(
		ctx, unbondingStakingEvent.StakingTxHashHex,
		unbondingStakingEvent.UnbondingStartHeight,
		unbondingStakingEvent.UnbondingTimeLock,
		types.UnbondingTxType,
	)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("Failed to process expire check")
		return err
	}

	// Save the unbonding staking delegation. This is the final step in the unbonding staking event processing
	// Please refer to the README.md for the details on the unbonding staking event processing workflow
	transitionErr := h.Services.TransitionToUnbondingState(
		ctx, unbondingStakingEvent.StakingTxHashHex, unbondingStakingEvent.UnbondingStartHeight,
		unbondingStakingEvent.UnbondingTimeLock, unbondingStakingEvent.UnbondingOutputIndex,
		unbondingStakingEvent.UnbondingTxHex, unbondingStakingEvent.UnbondingStartTimestamp,
	)
	if transitionErr != nil {
		log.Ctx(ctx).Err(transitionErr).Msg("Failed to transition to unbonding state")
		return transitionErr
	}

	return nil
}
