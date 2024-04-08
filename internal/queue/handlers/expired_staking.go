package handlers

import (
	"context"
	"encoding/json"

	"github.com/babylonchain/staking-api-service/internal/types"
	"github.com/babylonchain/staking-api-service/internal/utils"
	queueClient "github.com/babylonchain/staking-queue-client/client"
	"github.com/rs/zerolog/log"
)

func (h *QueueHandler) ExpiredStakingHandler(ctx context.Context, messageBody string) error {
	var expiredStakingEvent queueClient.ExpiredStakingEvent
	err := json.Unmarshal([]byte(messageBody), &expiredStakingEvent)
	if err != nil {
		log.Error().Err(err).Msg("Failed to unmarshal the message body into expiredStakingEvent")
		return err
	}

	// Check if the delegation is in the right state to process the unbonded(timelock expire) event
	state, stateErr := h.Services.GetDelegationState(ctx, expiredStakingEvent.StakingTxHashHex)
	// Requeue if found any error. Including not found error
	if stateErr != nil {
		return stateErr
	}
	if utils.Contains[types.DelegationState](utils.OutdatedStatesForUnbonded, state) {
		// Ignore the message as the delegation state already passed the unbonded state. This is an outdated duplication
		return nil
	}

	transitionErr := h.Services.TransitionToUnbondedState(ctx, expiredStakingEvent.TxType.ToString(), expiredStakingEvent.StakingTxHashHex)
	if transitionErr != nil {
		log.Error().Err(transitionErr).Str("StakingTxHashHex", expiredStakingEvent.StakingTxHashHex).Msg("Failed to transition to unbonded state after timelock expired")
		return transitionErr
	}

	return nil
}
