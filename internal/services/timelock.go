package services

import (
	"context"
	"net/http"

	"github.com/babylonchain/staking-api-service/internal/db"
	"github.com/babylonchain/staking-api-service/internal/types"
	"github.com/babylonchain/staking-api-service/internal/utils"
	"github.com/rs/zerolog/log"
)

// ProcessExpireCheck checks if the staking delegation has expired and updates the database.
// This method tolerate duplicated calls on the same stakingTxHashHex.
func (s *Services) ProcessExpireCheck(
	ctx context.Context, stakingTxHashHex string,
	startHeight, timelock uint64, txType types.StakingTxType,
) error {
	expireHeight := startHeight + timelock
	err := s.DbClient.SaveTimeLockExpireCheck(
		ctx, stakingTxHashHex, expireHeight, txType.ToString(),
	)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("Failed to save expire check")
		return types.NewInternalServiceError(err)
	}
	return nil
}

// TransitionToUnbondedState transitions the staking delegation to unbonded state.
// It returns true if the delegation is found and successfully transitioned to unbonded state.
func (s *Services) TransitionToUnbondedState(
	ctx context.Context, stakingType types.StakingTxType, stakingTxHashHex string,
) *types.Error {
	err := s.DbClient.TransitionToUnbondedState(ctx, stakingTxHashHex, utils.QualifiedStatesToUnbonded(stakingType))
	if err != nil {
		// If the delegation is not found, we can ignore the error, it just means the delegation is not in a state that we can transition to unbonded
		if db.IsNotFoundError(err) {
			errMsg := "delegation not found or no longer eligible to be unbonded after timelock expired"
			log.Ctx(ctx).Warn().Str("stakingTxHashHex", stakingTxHashHex).Err(err).Msg(errMsg)
			return types.NewErrorWithMsg(http.StatusForbidden, types.NotFound, errMsg)
		}
		log.Ctx(ctx).Err(err).Str("stakingTxHash", stakingTxHashHex).Msg("Failed to transition to unbonded state")
		return types.NewInternalServiceError(err)
	}
	return nil

}
