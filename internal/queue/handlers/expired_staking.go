package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/babylonchain/staking-api-service/internal/types"
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

	// If delegation does not exist in our system, then this message is out of order and should be retried later
	if exist, err := h.Services.IsDelegationPresent(ctx, expiredStakingEvent.StakingTxHashHex); err != nil {
		log.Err(err).Msg("Failed to check if delegation exists")
		return err
	} else if !exist {
		log.Warn().Str("stakingTxHash", expiredStakingEvent.StakingTxHashHex).Msg("Staking delegation not found, the expired message will need to be retried as it's out of order")
		return types.NewErrorWithMsg(http.StatusNotFound, types.NotFound, "Staking delegation not found")
	}

	_, err = h.Services.TransitionToUnbondedState(ctx, expiredStakingEvent.TxType.ToString(), expiredStakingEvent.StakingTxHashHex)
	if err != nil {
		log.Err(err).Msg("Failed to transition to unbonded state")
		return err
	}

	return nil
}
