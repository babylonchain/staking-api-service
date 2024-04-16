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
		log.Ctx(ctx).Error().Err(err).Msg("Failed to unmarshal the message body into expiredStakingEvent")
		return err
	}

	// Check if the delegation is in the right state to process the unbonded(timelock expire) event
	del, delErr := h.Services.GetDelegation(ctx, expiredStakingEvent.StakingTxHashHex)
	// Requeue if found any error. Including not found error
	if delErr != nil {
		return delErr
	}
	if utils.Contains[types.DelegationState](utils.OutdatedStatesForUnbonded(), del.State) {
		log.Ctx(ctx).Warn().Str("StakingTxHashHex", expiredStakingEvent.StakingTxHashHex).Msg("delegation state is outdated for unbonded event")
		// Ignore the message as the delegation state already passed the unbonded state. This is an outdated duplication
		return nil
	}

	txType, err := types.StakingTxTypeFromString(expiredStakingEvent.TxType)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Str("TxType", expiredStakingEvent.TxType).Msg("Failed to convert TxType from string")
		return err
	}

	// Perform the stats calculation
	statsErr := h.Services.ProcessStakingStatsCalculation(
		ctx, expiredStakingEvent.StakingTxHashHex,
		del.StakerPkHex,
		del.FinalityProviderPkHex,
		types.Unbonded,
		del.StakingValue,
	)
	if statsErr != nil {
		log.Error().Err(statsErr).Msg("Failed to process staking stats calculation after timelock expired")
		return statsErr
	}

	transitionErr := h.Services.TransitionToUnbondedState(ctx, txType, expiredStakingEvent.StakingTxHashHex)
	if transitionErr != nil {
		log.Ctx(ctx).Error().Err(transitionErr).Str("StakingTxHashHex", expiredStakingEvent.StakingTxHashHex).Msg("Failed to transition to unbonded state after timelock expired")
		return transitionErr
	}

	return nil
}
