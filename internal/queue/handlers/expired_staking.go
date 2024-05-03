package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/babylonchain/staking-api-service/internal/types"
	"github.com/babylonchain/staking-api-service/internal/utils"
	queueClient "github.com/babylonchain/staking-queue-client/client"
	"github.com/rs/zerolog/log"
)

func (h *QueueHandler) ExpiredStakingHandler(ctx context.Context, messageBody string) *types.Error {
	var expiredStakingEvent queueClient.ExpiredStakingEvent
	err := json.Unmarshal([]byte(messageBody), &expiredStakingEvent)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("Failed to unmarshal the message body into expiredStakingEvent")
		return types.NewError(http.StatusBadRequest, types.BadRequest, err)
	}

	// Check if the delegation is in the right state to process the unbonded(timelock expire) event
	del, delErr := h.Services.GetDelegation(ctx, expiredStakingEvent.StakingTxHashHex)
	// Requeue if found any error. Including not found error
	if delErr != nil {
		return delErr
	}
	if utils.Contains[types.DelegationState](utils.OutdatedStatesForUnbonded(), del.State) {
		// Ignore the message as the delegation state already passed the unbonded state. This is an outdated duplication
		log.Ctx(ctx).Debug().Str("StakingTxHashHex", expiredStakingEvent.StakingTxHashHex).
			Msg("delegation state is outdated for unbonded event")
		return nil
	}

	txType, err := types.StakingTxTypeFromString(expiredStakingEvent.TxType)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Str("TxType", expiredStakingEvent.TxType).Msg("Failed to convert TxType from string")
		return types.NewError(http.StatusBadRequest, types.BadRequest, err)
	}

	// Perform the async stats calculation by emit the stats event
	statsError := h.EmitStatsEvent(ctx, queueClient.NewStatsEvent(
		expiredStakingEvent.StakingTxHashHex,
		del.StakerPkHex,
		del.FinalityProviderPkHex,
		del.StakingValue,
		types.Unbonded.ToString(),
	))
	if statsError != nil {
		log.Ctx(ctx).Error().Err(statsError).Msg("Failed to emit stats event after timelock expired")
		return statsError
	}

	transitionErr := h.Services.TransitionToUnbondedState(ctx, txType, expiredStakingEvent.StakingTxHashHex)
	if transitionErr != nil {
		return transitionErr
	}

	return nil
}
