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

func (h *QueueHandler) WithdrawStakingHandler(ctx context.Context, messageBody string) error {
	var withdrawnStakingEvent queueClient.WithdrawStakingEvent
	err := json.Unmarshal([]byte(messageBody), &withdrawnStakingEvent)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("Failed to unmarshal the message body into withdrawnStakingEvent")
		return err
	}

	// Check if the delegation is in the right state to process the withdrawn event.
	state, stateErr := h.Services.GetDelegationState(ctx, withdrawnStakingEvent.StakingTxHashHex)
	// Requeue if found any error. Including not found error
	if stateErr != nil {
		return stateErr
	}
	// Requeue if the current state is not in the qualified states to transition to withdrawn
	// We will wait for the unbonded message to be processed first.
	if !utils.Contains(utils.QualifiedStatesToWithdraw(), state) {
		errMsg := "delegation is not in the qualified state to transition to withdrawn"
		log.Ctx(ctx).Warn().Str("state", state.ToString()).Msg(errMsg)
		return types.NewErrorWithMsg(http.StatusForbidden, types.Forbidden, errMsg)
	}

	if utils.Contains(utils.OutdatedStatesForWithdraw(), state) {
		// Ignore the message as the delegation state is withdrawn. Nothing to do anymore
		return nil
	}

	// Perform stats calculation
	if err := h.Services.ProcessStakingStatsCalculation(ctx, withdrawnStakingEvent); err != nil {
		log.Ctx(ctx).Err(err).Msg("Failed to update stats while processing withdrawn staking event")
		return err
	}

	// Transition to withdrawn state
	// Please refer to the README.md for the details on the event processing workflow
	transitionErr := h.Services.TransitionToWithdrawnState(
		ctx, withdrawnStakingEvent.StakingTxHashHex,
	)
	if transitionErr != nil {
		log.Ctx(ctx).Err(transitionErr).Msg("Failed to transition to withdrawn state")
		return transitionErr
	}

	return nil
}
