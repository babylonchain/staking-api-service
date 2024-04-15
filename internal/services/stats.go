package services

import (
	"context"
	"fmt"
	"net/http"

	"github.com/babylonchain/staking-api-service/internal/db"
	"github.com/babylonchain/staking-api-service/internal/types"
	"github.com/rs/zerolog/log"
)

// ProcessStakingStatsCalculation calculates the staking stats and updates the database.
// This method tolerate duplicated calls, only the first call will be processed.
func (s *Services) ProcessStakingStatsCalculation(
	ctx context.Context, stakingTxHashHex, stakerPkHex, fpPkHex string,
	txType types.StakingTxType, amount uint64,
) error {
	// Fetch existing or initialize the stats lock document if not exist
	statsLockDocument, err := s.DbClient.GetOrCreateStatsLock(ctx, stakingTxHashHex, txType.ToString())
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Str("stakingTxHashHex", stakingTxHashHex).Msg("error while fetching stats lock document")
		return types.NewInternalServiceError(err)
	}
	switch txType {
	case types.ActiveTxType:
		// Add to the overall stats
		if !statsLockDocument.OverallStats {
			err = s.DbClient.IncrementOverallStats(ctx, stakingTxHashHex, int64(amount))
			if err != nil {
				if db.IsNotFoundError(err) {
					// This is a duplicate call, ignore it
					return nil
				}
				log.Ctx(ctx).Error().Err(err).Str("stakingTxHashHex", stakingTxHashHex).
					Msg("error while incrementing overall stats")
				return types.NewInternalServiceError(err)
			}
		}
	case types.UnbondingTxType:
		// Subtract from the overall stats
		if !statsLockDocument.OverallStats {
			err = s.DbClient.SubtractOverallStats(ctx, stakingTxHashHex, -int64(amount))
			if err != nil {
				if db.IsNotFoundError(err) {
					// This is a duplicate call, ignore it
					return nil
				}
				log.Ctx(ctx).Error().Err(err).Str("stakingTxHashHex", stakingTxHashHex).
					Msg("error while subtracting overall stats")
				return types.NewInternalServiceError(err)
			}
		}
		// Subtract from the finality stats
	default:
		return types.NewErrorWithMsg(
			http.StatusForbidden,
			types.Forbidden,
			fmt.Sprintf("staking tx type not regonised for performing stats calculation: %s", txType.ToString()),
		)
	}
	return nil
}
