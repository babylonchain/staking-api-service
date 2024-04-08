package db

import (
	"context"

	"github.com/babylonchain/staking-api-service/internal/db/model"
	"github.com/babylonchain/staking-api-service/internal/types"
)

type DBClient interface {
	Ping(ctx context.Context) error
	SaveActiveStakingDelegation(
		ctx context.Context, stakingTxHashHex, stakerPkHex, fpPkHex string, stakingTxHex string,
		amount, startHeight, timelock, outputIndex uint64, startTimestamp string,
	) error
	FindDelegationsByStakerPk(
		ctx context.Context, stakerPk string, paginationToken string,
	) (*DbResultMap[model.DelegationDocument], error)
	SaveUnbondingTx(
		ctx context.Context, stakingTxHashHex, unbondingTxHashHex, txHex, signatureHex string,
	) error
	FindDelegationByTxHashHex(ctx context.Context, txHashHex string) (*model.DelegationDocument, error)
	SaveTimeLockExpireCheck(ctx context.Context, stakingTxHashHex string, expireHeight uint64, txType string) error
	FindFinalityProvidersByPkHex(ctx context.Context, pkHex []string) (map[string]model.FinalityProviderDocument, error)
	SaveUnprocessableMessage(ctx context.Context, messageBody, receipt string) error
	TransitionToUnbondedState(
		ctx context.Context, stakingTxHashHex string, eligiblePreviousState []types.DelegationState,
	) error
	TransitionToUnbondingState(
		ctx context.Context, txHashHex string, startHeight, timelock, outputIndex uint64, txHex, startTimestamp string,
	) error
}
