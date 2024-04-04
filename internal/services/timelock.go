package services

import (
	"context"
	"net/http"

	"github.com/babylonchain/staking-api-service/internal/db"
	"github.com/babylonchain/staking-api-service/internal/types"
	"github.com/rs/zerolog/log"
)

// ProcessExpireCheck checks if the staking delegation has expired and updates the database.
// This method tolerate duplicated calls on the same stakingTxHashHex.
func (s *Services) ProcessExpireCheck(
	ctx context.Context, stakingTxHashHex string,
	startHeight, timelock uint64, txType string,
) error {
	expireHeight := startHeight + timelock
	err := s.DbClient.SaveTimeLockExpireCheck(
		ctx, stakingTxHashHex, expireHeight, txType,
	)
	if err != nil {
		log.Err(err).Msg("Failed to save expire check")
		return types.NewInternalServiceError(err)
	}
	return nil
}

type ExpiredTxType string

const (
	ActiveType    ExpiredTxType = "active"
	UnbondingType ExpiredTxType = "unbonding"
)

func (s ExpiredTxType) ToString() string {
	return string(s)
}

func ExpiredTxTypeFromString(s string) ExpiredTxType {
	switch s {
	case ActiveType.ToString():
		return ActiveType
	case UnbondingType.ToString():
		return UnbondingType
	default:
		return ""
	}
}

// TransitionToUnbondedState transitions the staking delegation to unbonded state.
// It returns true if the delegation is found and successfully transitioned to unbonded state.
func (s *Services) TransitionToUnbondedState(
	ctx context.Context, stakingType, stakingTxHashHex string,
) (bool, error) {
	switch ExpiredTxTypeFromString(stakingType) {
	case ActiveType:
		err := s.DbClient.TransitionState(ctx, stakingTxHashHex, types.Unbonded.ToString(), []string{types.Active.ToString()})
		if err != nil {
			// If the delegation is not found, we can ignore the error, it just means the delegation is not in a state that we can transition to unbonded
			if db.IsNotFoundError(err) {
				log.Warn().Str("stakingTxHash", stakingTxHashHex).Msg("Staking delegation not found")
				return false, nil
			}
			log.Err(err).Str("stakingTxHash", stakingTxHashHex).Msg("Failed to transition to unbonded state")
			return false, types.NewInternalServiceError(err)
		}
		return true, nil
	case UnbondingType:
		// TODO: Process the unbonding staking event https://github.com/babylonchain/staking-api-service/issues/8
		log.Info().Str("stakingTxHash", stakingTxHashHex).Msg("Processing unbonding staking event")
		return false, types.NewErrorWithMsg(http.StatusNotImplemented, types.Forbidden, "Unbonding staking event is not implemented yet")
	default:
		log.Error().Str("stakingTxHash", stakingTxHashHex).Msg("Unknown staking type")
		return false, types.NewErrorWithMsg(http.StatusInternalServerError, types.InternalServiceError, "Unknown staking type received in the message body")
	}
}
