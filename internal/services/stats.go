package services

import (
	"context"
	"fmt"
	"net/http"

	"github.com/babylonchain/staking-api-service/internal/db"
	"github.com/babylonchain/staking-api-service/internal/types"
	"github.com/rs/zerolog/log"
)

type StatsPublic struct {
	ActiveTvl         int64 `json:"active_tvl"`
	TotalTvl          int64 `json:"total_tvl"`
	ActiveDelegations int64 `json:"active_delegations"`
	TotalDelegations  int64 `json:"total_delegations"`
	TotalStakers      int64 `json:"total_stakers"`
}

// ProcessStakingStatsCalculation calculates the staking stats and updates the database.
// This method tolerate duplicated calls, only the first call will be processed.
func (s *Services) ProcessStakingStatsCalculation(
	ctx context.Context, stakingTxHashHex, stakerPkHex, fpPkHex string,
	state types.DelegationState, amount uint64,
) *types.Error {
	// Fetch existing or initialize the stats lock document if not exist
	statsLockDocument, err := s.DbClient.GetOrCreateStatsLock(ctx, stakingTxHashHex, state.ToString())
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Str("stakingTxHashHex", stakingTxHashHex).Msg("error while fetching stats lock document")
		return types.NewInternalServiceError(err)
	}
	switch state {
	case types.Active:
		// Add to the overall stats
		if !statsLockDocument.OverallStats {
			err = s.DbClient.IncrementOverallStats(ctx, stakingTxHashHex, amount)
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
		if !statsLockDocument.FinalityStats {
			// Add to the finality stats
			err = s.DbClient.IncrementFinalityProviderStats(ctx, stakingTxHashHex, fpPkHex, amount)
			if err != nil {
				if db.IsNotFoundError(err) {
					return nil
				}
				log.Ctx(ctx).Error().Err(err).Str("stakingTxHashHex", stakingTxHashHex).
					Msg("error while incrementing finality stats")
				return types.NewInternalServiceError(err)
			}
		}
	case types.Unbonded:
		// Subtract from the overall stats
		if !statsLockDocument.OverallStats {
			err = s.DbClient.SubtractOverallStats(ctx, stakingTxHashHex, amount)
			if err != nil {
				if db.IsNotFoundError(err) {
					return nil
				}
				log.Ctx(ctx).Error().Err(err).Str("stakingTxHashHex", stakingTxHashHex).
					Msg("error while subtracting overall stats")
				return types.NewInternalServiceError(err)
			}
		}
		// Subtract from the finality stats
		if !statsLockDocument.FinalityStats {
			err = s.DbClient.SubtractFinalityProviderStats(ctx, stakingTxHashHex, fpPkHex, amount)
			if err != nil {
				if db.IsNotFoundError(err) {
					return nil
				}
				log.Ctx(ctx).Error().Err(err).Str("stakingTxHashHex", stakingTxHashHex).
					Msg("error while subtracting finality stats")
				return types.NewInternalServiceError(err)
			}
		}
	default:
		return types.NewErrorWithMsg(
			http.StatusForbidden,
			types.Forbidden,
			fmt.Sprintf("invalid delegation state for stats calculation: %s", state),
		)
	}
	return nil
}

func (s *Services) GetOverallStats(ctx context.Context) (*StatsPublic, *types.Error) {
	stats, err := s.DbClient.GetOverallStats(ctx)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("error while fetching overall stats")
		return nil, types.NewInternalServiceError(err)
	}

	return &StatsPublic{
		ActiveTvl:         stats.ActiveTvl,
		TotalTvl:          stats.TotalTvl,
		ActiveDelegations: stats.ActiveDelegations,
		TotalDelegations:  stats.TotalDelegations,
		TotalStakers:      0, // TODO: Staker stats is yet to be implemented
	}, nil
}
