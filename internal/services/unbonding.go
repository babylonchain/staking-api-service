package services

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"net/http"

	"github.com/babylonchain/babylon/btcstaking"
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/rs/zerolog/log"

	"github.com/babylonchain/staking-api-service/internal/db"
	"github.com/babylonchain/staking-api-service/internal/types"
	"github.com/babylonchain/staking-api-service/internal/utils"
)

func (s *Services) verifyUnbondingRequestSignature(ctx context.Context, stakingTxHashHex, txHex, signatureHex string) error {
	// 1. check the existence of the relevant delegation
	delegationDoc, err := s.DbClient.FindDelegationByTxHashHex(ctx, stakingTxHashHex)
	if err != nil {
		if ok := db.IsNotFoundError(err); ok {
			log.Warn().Err(err).Msg("delegation not found, hence not eligible for unbonding")
			return types.NewErrorWithMsg(http.StatusForbidden, types.NotFound, "delegation not found")
		}
		log.Error().Err(err).Msg("error while fetching delegation")
		return types.NewError(http.StatusInternalServerError, types.InternalServiceError, err)
	}

	// 2. validate that un-bonding transaction has proper shape
	invalidUnbondingTxMsg := "invalid unbonding tx"
	unbondingTx, err := utils.GetBtcTxFromHex(txHex)
	if err != nil {
		log.Error().Err(err).Msg(invalidUnbondingTxMsg)
		return types.NewError(http.StatusInternalServerError, types.InternalServiceError, err)
	}
	if len(unbondingTx.TxIn) != 1 {
		inputErr := fmt.Errorf("unbonding tx must have 1 input, got %d", len(unbondingTx.TxIn))
		log.Error().Err(inputErr).Msg(invalidUnbondingTxMsg)
		return types.NewError(http.StatusInternalServerError, types.InternalServiceError, inputErr)
	}
	if len(unbondingTx.TxOut) != 1 {
		outputErr := fmt.Errorf("unbonding tx must have 1 output, got %d", len(unbondingTx.TxOut))
		log.Error().Err(outputErr).Msg(invalidUnbondingTxMsg)
		return types.NewError(http.StatusInternalServerError, types.InternalServiceError, outputErr)
	}
	if unbondingTx.LockTime != 0 {
		lockTimeErr := fmt.Errorf("unbonding tx must have lock time equal to 0, got %d", unbondingTx.LockTime)
		log.Error().Err(lockTimeErr).Msg(invalidUnbondingTxMsg)
		return types.NewError(http.StatusInternalServerError, types.InternalServiceError, lockTimeErr)
	}

	// 3. validate the un-bonding transaction points to the previous staking tx
	stakingTxHash, err := chainhash.NewHashFromStr(delegationDoc.StakingTxHashHex)
	if err != nil {
		log.Error().Err(err).Msg("error while decoding staking tx hash from hex")
		return types.NewError(http.StatusInternalServerError, types.InternalServiceError, err)
	}
	if !unbondingTx.TxIn[0].PreviousOutPoint.Hash.IsEqual(stakingTxHash) {
		inputErr := fmt.Errorf("the unbonding tx input must match the previous staking tx hash")
		log.Error().Err(inputErr).Msg(invalidUnbondingTxMsg)
		return types.NewError(http.StatusInternalServerError, types.InternalServiceError, inputErr)
	}
	if uint64(unbondingTx.TxIn[0].PreviousOutPoint.Index) != delegationDoc.StakingOutputIndex {
		inputErr := fmt.Errorf("the unbonding tx input must match the previous staking tx output index")
		log.Error().Err(inputErr).Msg(invalidUnbondingTxMsg)
		return types.NewError(http.StatusInternalServerError, types.InternalServiceError, inputErr)
	}

	// 4. validate that output commits to proper taproot script tree
	params := s.GetGlobalParams()
	stakerPk, err := utils.GetBtcPkFromHex(delegationDoc.StakerPkHex)
	if err != nil {
		log.Error().Err(err).Msg("error while parsing staker public key from hex")
		return types.NewError(http.StatusInternalServerError, types.InternalServiceError, err)
	}
	finalityProviderPk, err := utils.GetBtcPkFromHex(delegationDoc.FinalityProviderPkHex)
	if err != nil {
		log.Error().Err(err).Msg("error while parsing finality provider public key from hex")
		return types.NewError(http.StatusInternalServerError, types.InternalServiceError, err)
	}
	expectedUnbondingOutput, err := btcstaking.BuildUnbondingInfo(
		stakerPk,
		[]*btcec.PublicKey{finalityProviderPk},
		params.CovenantPks,
		uint32(params.CovenantQuorum),
		uint16(params.UnbondingTime),
		// unbondingAmount does not affect taproot script
		0,
		// TODO should parameterize BTC net in config
		&chaincfg.RegressionNetParams,
	)
	if err != nil {
		log.Error().Err(err).Msg("error while building unbonding info")
		return types.NewError(http.StatusInternalServerError, types.InternalServiceError, err)
	}
	if !bytes.Equal(unbondingTx.TxOut[0].PkScript, expectedUnbondingOutput.UnbondingOutput.PkScript) {
		outputErr := fmt.Errorf("invalid unbonding output script")
		log.Error().Err(err).Msg(invalidUnbondingTxMsg)
		return types.NewError(http.StatusInternalServerError, types.InternalServiceError, outputErr)
	}

	// 5. verify the signature
	stakingInfo, err := btcstaking.BuildStakingInfo(
		stakerPk,
		[]*btcec.PublicKey{finalityProviderPk},
		params.CovenantPks,
		uint32(params.CovenantQuorum),
		uint16(delegationDoc.StakingTimeLock),
		btcutil.Amount(delegationDoc.StakingValue),
		// TODO should parameterize BTC net in config
		&chaincfg.RegressionNetParams,
	)
	if err != nil {
		log.Error().Err(err).Msg("error while building staking info")
		return types.NewError(http.StatusInternalServerError, types.InternalServiceError, err)
	}
	sigBytes, err := hex.DecodeString(signatureHex)
	if err != nil {
		log.Error().Err(err).Msg("error while decoding signature from hex")
		return types.NewError(http.StatusInternalServerError, types.InternalServiceError, err)
	}
	unbondingSpendInfo, err := stakingInfo.UnbondingPathSpendInfo()
	if err != nil {
		log.Error().Err(err).Msg("error while getting unbonding path spend info")
		return types.NewError(http.StatusInternalServerError, types.InternalServiceError, err)
	}
	if err := btcstaking.VerifyTransactionSigWithOutputData(
		unbondingTx,
		stakingInfo.StakingOutput.PkScript,
		stakingInfo.StakingOutput.Value,
		unbondingSpendInfo.GetPkScriptPath(),
		stakerPk,
		sigBytes,
	); err != nil {
	}

	// 6. validate that un-bonding output has most of the value of staking output
	// TODO global parameter TBD

	return nil
}

func (s *Services) UnbondDelegation(ctx context.Context, stakingTxHashHex, unbondingTxHashHex, txHex, signatureHex string) *types.Error {
	err := s.verifyUnbondingRequestSignature(ctx, stakingTxHashHex, txHex, signatureHex)
	if err != nil {
		log.Warn().Err(err).Msg("did not pass unbonding request verification")
		return types.NewError(http.StatusForbidden, types.ValidationError, err)
	}

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
