package utils

import (
	"bytes"
	"encoding/hex"
	"fmt"

	"github.com/babylonchain/babylon/btcstaking"
	bbntypes "github.com/babylonchain/babylon/types"
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"

	"github.com/babylonchain/staking-api-service/internal/types"
)

func GetBtcPkFromHex(pkHex string) (*btcec.PublicKey, error) {
	pkBytes, err := hex.DecodeString(pkHex)
	if err != nil {
		return nil, err
	}

	return schnorr.ParsePubKey(pkBytes)
}

func GetBtcPksFromStrings(pkStrings []string) ([]*btcec.PublicKey, error) {
	pks := make([]*btcec.PublicKey, len(pkStrings))
	for i, pkStr := range pkStrings {
		pk, err := GetBtcPkFromHex(pkStr)
		if err != nil {
			return nil, err
		}
		pks[i] = pk
	}

	return pks, nil
}

func VerifyUnbondingRequest(
	stakingTxHashHex,
	unbondingTxHex,
	stakerPkHex,
	finalityProviderPkHex,
	unbondingSigHex string,
	stakingTimeLock,
	stakingOutputIndex,
	stakingValue uint64,
	params *types.GlobalParams,
	btcNetParam *chaincfg.Params,
) error {
	// 1. validate that un-bonding transaction has proper shape
	unbondingTx, _, err := bbntypes.NewBTCTxFromHex(unbondingTxHex)
	if err != nil {
		return fmt.Errorf("failed to decode unbonding tx from hex: %w", err)
	}
	if len(unbondingTx.TxIn) != 1 {
		return fmt.Errorf("unbonding tx must have 1 input, got %d", len(unbondingTx.TxIn))
	}
	if len(unbondingTx.TxOut) != 1 {
		return fmt.Errorf("unbonding tx must have 1 output, got %d", len(unbondingTx.TxOut))
	}
	if unbondingTx.LockTime != 0 {
		return fmt.Errorf("unbonding tx must have lock time equal to 0, got %d", unbondingTx.LockTime)
	}

	// 2. validate the un-bonding transaction points to the previous staking tx
	stakingTxHash, err := chainhash.NewHashFromStr(stakingTxHashHex)
	if err != nil {
		return fmt.Errorf("failed to decode staking tx hash from hex: %w", err)
	}
	if !unbondingTx.TxIn[0].PreviousOutPoint.Hash.IsEqual(stakingTxHash) {
		return fmt.Errorf("the unbonding tx input must match the previous staking tx hash, expected: %s, got: %s",
			stakingTxHashHex,
			unbondingTx.TxIn[0].PreviousOutPoint.Hash.String(),
		)
	}
	if uint64(unbondingTx.TxIn[0].PreviousOutPoint.Index) != stakingOutputIndex {
		return fmt.Errorf("the unbonding tx input must match the previous staking tx output index, expected: %d, got: %d",
			stakingOutputIndex,
			unbondingTx.TxIn[0].PreviousOutPoint.Index,
		)
	}

	// 3. validate that output commits to proper taproot script tree
	covenantPks, err := GetBtcPksFromStrings(params.CovenantPks)
	if err != nil {
		return fmt.Errorf("failed to decode coveant public keys from strings: %w", err)
	}

	stakerPk, err := GetBtcPkFromHex(stakerPkHex)
	if err != nil {
		return fmt.Errorf("failed to decode staker public key from hex: %w", err)
	}

	finalityProviderPk, err := GetBtcPkFromHex(finalityProviderPkHex)
	if err != nil {
		return fmt.Errorf("failed to decode finality provider public key from hex: %w", err)
	}

	expectedUnbondingOutput, err := btcstaking.BuildUnbondingInfo(
		stakerPk,
		[]*btcec.PublicKey{finalityProviderPk},
		covenantPks,
		uint32(params.CovenantQuorum),
		uint16(params.UnbondingTime),
		// unbondingAmount does not affect taproot script
		0,
		btcNetParam,
	)
	if err != nil {
		return fmt.Errorf("failed to build unbonding info")
	}

	if !bytes.Equal(unbondingTx.TxOut[0].PkScript, expectedUnbondingOutput.UnbondingOutput.PkScript) {
		return fmt.Errorf("invalid unbonding output script")
	}

	// 5. verify the signature
	stakingInfo, err := btcstaking.BuildStakingInfo(
		stakerPk,
		[]*btcec.PublicKey{finalityProviderPk},
		covenantPks,
		uint32(params.CovenantQuorum),
		uint16(stakingTimeLock),
		btcutil.Amount(stakingValue),
		btcNetParam,
	)
	if err != nil {
		return fmt.Errorf("failed to build staking info")
	}
	sigBytes, err := hex.DecodeString(unbondingSigHex)
	if err != nil {
		return fmt.Errorf("failed to decode unbonding signature from hex")
	}
	unbondingSpendInfo, err := stakingInfo.UnbondingPathSpendInfo()
	if err != nil {
		return fmt.Errorf("failed to build unbonding path spend info")
	}
	if err := btcstaking.VerifyTransactionSigWithOutputData(
		unbondingTx,
		stakingInfo.StakingOutput.PkScript,
		stakingInfo.StakingOutput.Value,
		unbondingSpendInfo.GetPkScriptPath(),
		stakerPk,
		sigBytes,
	); err != nil {
		return fmt.Errorf("invalid unbonding signature")
	}

	// 6. validate that un-bonding output has most of the value of staking output
	// TODO global parameter TBD

	return nil
}

func GetBtcNetParamesFromString(net string) (*chaincfg.Params, error) {
	var netParams chaincfg.Params
	switch net {
	case "mainnet":
		netParams = chaincfg.MainNetParams
	case "testnet3":
		netParams = chaincfg.TestNet3Params
	case "regtest":
		netParams = chaincfg.RegressionNetParams
	case "simnet":
		netParams = chaincfg.SimNetParams
	case "signet":
		netParams = chaincfg.SigNetParams
	default:
		return nil, fmt.Errorf("invalid network: %s", net)
	}
	return &netParams, nil
}
