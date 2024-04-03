package services

import (
	"context"
	"net/http"

	"github.com/babylonchain/staking-api-service/internal/db"
	"github.com/babylonchain/staking-api-service/internal/db/model"
	"github.com/babylonchain/staking-api-service/internal/types"
	"github.com/rs/zerolog/log"
)

type TransactionPublic struct {
	TxHex          string `json:"tx_hex"`
	OutputIndex    uint64 `json:"output_index"`
	StartTimestamp string `json:"start_timestamp"`
	StartHeight    uint64 `json:"start_height"`
	TimeLock       uint64 `json:"timelock"`
}

type DelegationPublic struct {
	StakingTxHashHex      string             `json:"staking_tx_hash_hex"`
	StakerPkHex           string             `json:"staker_pk_hex"`
	FinalityProviderPkHex string             `json:"finality_provider_pk_hex"`
	State                 string             `json:"state"`
	StakingValue          uint64             `json:"staking_value"`
	StakingTx             *TransactionPublic `json:"staking_tx"`
	UnbondingTx           *TransactionPublic `json:"unbonding_tx,omitempty"`
}

func fromDelegationDocument(d model.DelegationDocument) DelegationPublic {
	delPublic := DelegationPublic{
		StakingTxHashHex:      d.StakingTxHashHex,
		StakerPkHex:           d.StakerPkHex,
		FinalityProviderPkHex: d.FinalityProviderPkHex,
		StakingValue:          d.StakingValue,
		State:                 d.State.ToString(),
		StakingTx: &TransactionPublic{
			TxHex:          d.StakingTx.TxHex,
			OutputIndex:    d.StakingTx.OutputIndex,
			StartTimestamp: d.StakingTx.StartTimestamp,
			StartHeight:    d.StakingTx.StartHeight,
			TimeLock:       d.StakingTx.TimeLock,
		},
	}

	// Add unbonding transaction if it exists
	if d.UnbondingTx != nil && d.UnbondingTx.TxHex != "" {
		delPublic.UnbondingTx = &TransactionPublic{
			TxHex:          d.UnbondingTx.TxHex,
			OutputIndex:    d.UnbondingTx.OutputIndex,
			StartTimestamp: d.UnbondingTx.StartTimestamp,
			StartHeight:    d.UnbondingTx.StartHeight,
			TimeLock:       d.UnbondingTx.TimeLock,
		}
	}
	return delPublic
}

func (s *Services) DelegationsByStakerPk(ctx context.Context, stakerPk string, pageToken string) ([]DelegationPublic, string, *types.Error) {
	resultMap, err := s.DbClient.FindDelegationsByStakerPk(ctx, stakerPk, pageToken)
	if err != nil {
		if db.IsInvalidPaginationTokenError(err) {
			log.Warn().Err(err).Msg("Invalid pagination token when fetching delegations by staker pk")
			return nil, "", types.NewError(http.StatusBadRequest, types.BadRequest, err)
		}
		log.Error().Err(err).Msg("Failed to find delegations by staker pk")
		return nil, "", types.NewInternalServiceError(err)
	}
	var delegations []DelegationPublic
	for _, d := range resultMap.Data {
		delegations = append(delegations, fromDelegationDocument(d))
	}
	return delegations, resultMap.PaginationToken, nil
}

// SaveActiveStakingDelegation saves the active staking delegation to the database.
func (s *Services) SaveActiveStakingDelegation(
	ctx context.Context, txHashHex, stakerPkHex, finalityProviderPkHex string,
	value, startHeight uint64, stakingTimestamp string, timeLock, stakingOutputIndex uint64,
	stakingTxHex string,
) error {
	err := s.DbClient.SaveActiveStakingDelegation(
		ctx, txHashHex, stakerPkHex, finalityProviderPkHex, stakingTxHex,
		value, startHeight, timeLock, stakingOutputIndex, stakingTimestamp,
	)
	if err != nil {
		if ok := db.IsDuplicateKeyError(err); ok {
			log.Warn().Err(err).Msg("Skip the active staking event as it already exists in the database")
			// TODO: Add metrics for duplicate active staking events
			return nil
		}
		log.Error().Err(err).Msg("Failed to save active staking delegation")
		return types.NewInternalServiceError(err)
	}
	return nil
}
