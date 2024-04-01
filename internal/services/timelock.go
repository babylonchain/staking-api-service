package services

import (
	"context"

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
