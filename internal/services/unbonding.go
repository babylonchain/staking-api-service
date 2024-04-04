package services

import (
	"context"
	"net/http"

	"github.com/rs/zerolog/log"

	"github.com/babylonchain/staking-api-service/internal/db"
	"github.com/babylonchain/staking-api-service/internal/types"
	"github.com/babylonchain/staking-api-service/internal/utils"
)

func (s *Services) UnbondDelegation(ctx context.Context, stakingTxHashHex, unbondingTxHashHex, txHex, signatureHex string) *types.Error {
	// 1. check the delegation is eligible for unbonding
	delegationDoc, err := s.DbClient.FindDelegationByTxHashHex(ctx, stakingTxHashHex)
	if err != nil {
		if ok := db.IsNotFoundError(err); ok {
			log.Warn().Err(err).Msg("delegation not found, hence not eligible for unbonding")
			return types.NewErrorWithMsg(http.StatusForbidden, types.NotFound, "delegation not found")
		}
		log.Error().Err(err).Msg("error while fetching delegation")
		return types.NewError(http.StatusInternalServerError, types.InternalServiceError, err)
	}

	if delegationDoc.State != types.Active {
		log.Warn().Msg("delegation state is not active, hence not eligible for unbonding")
		return types.NewErrorWithMsg(http.StatusForbidden, types.Forbidden, "delegation state is not active")
	}

	// 2. verify the unbonding request
	if err := utils.VerifyUnbondingRequest(
		delegationDoc.StakingTxHashHex,
		txHex,
		delegationDoc.StakerPkHex,
		delegationDoc.FinalityProviderPkHex,
		signatureHex,
		delegationDoc.StakingTx.TimeLock,
		delegationDoc.StakingTx.OutputIndex,
		delegationDoc.StakingValue,
		s.params,
		// TODO: BTC net param should be set in config
		"regtest",
	); err != nil {
		log.Warn().Err(err).Msg("did not pass unbonding request verification")
		return types.NewError(http.StatusForbidden, types.ValidationError, err)
	}

	// 3. save unbonding tx into DB
	err = s.DbClient.SaveUnbondingTx(ctx, stakingTxHashHex, unbondingTxHashHex, txHex, signatureHex)
	if err != nil {
		if ok := db.IsDuplicateKeyError(err); ok {
			log.Warn().Err(err).Msg("unbonding request already been submitted into the system")
			return types.NewError(http.StatusForbidden, types.Forbidden, err)
		} else if ok := db.IsNotFoundError(err); ok {
			log.Warn().Err(err).Msg("no active delegation found for unbonding request")
			return types.NewError(http.StatusForbidden, types.Forbidden, err)
		}
		log.Error().Err(err).Msg("failed to save unbonding tx")
		return types.NewError(http.StatusInternalServerError, types.InternalServiceError, err)
	}
	return nil
}

func (s *Services) IsEligibleForUnbonding(ctx context.Context, stakingTxHashHex string) *types.Error {
	delegationDoc, err := s.DbClient.FindDelegationByTxHashHex(ctx, stakingTxHashHex)
	if err != nil {
		if ok := db.IsNotFoundError(err); ok {
			log.Warn().Err(err).Msg("delegation not found, hence not eligible for unbonding")
			return types.NewErrorWithMsg(http.StatusForbidden, types.NotFound, "delegation not found")
		}
		log.Error().Err(err).Msg("error while fetching delegation")
		return types.NewError(http.StatusInternalServerError, types.InternalServiceError, err)
	}

	if delegationDoc.State != types.Active {
		log.Warn().Msg("delegation state is not active, hence not eligible for unbonding")
		return types.NewErrorWithMsg(http.StatusForbidden, types.Forbidden, "delegation state is not active")
	}
	return nil
}
