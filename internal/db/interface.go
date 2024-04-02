package db

import (
	"context"

	"github.com/babylonchain/staking-api-service/internal/db/model"
)

type DBClient interface {
	Ping(ctx context.Context) error
	SaveActiveStakingDelegation(
		ctx context.Context,
		stakingTxHashHex, stakerPkHex, finalityProviderPkHex string,
		amount, startHeight, timelock uint64,
	) error
	FindDelegationsByStakerPk(
		ctx context.Context, stakerPk string, paginationToken string,
	) (*DbResultMap[model.DelegationDocument], error)

	SaveUnbondingTx(
		ctx context.Context, stakingTxHashHex, unbondingTxHashHex, txHex, signatureHex string,
	) error

	FindDelegationByTxHashHex(ctx context.Context, txHashHex string) (*model.DelegationDocument, error)

	SaveTimeLockExpireCheck(ctx context.Context, stakingTxHashHex string, expireHeight uint64, txType string) error
}
