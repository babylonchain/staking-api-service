package db

import (
	"context"

	"github.com/babylonchain/staking-api-service/internal/db/model"
)

type DBClient interface {
	Ping(ctx context.Context) error
	SaveActiveStakingDelegation(
		ctx context.Context,
		stakingTxHashHex, stakerPhHex, finalityProviderPkHex string,
		amount, startHeight, timelock uint64,
	) error
	FindDelegationsByStakerPk(
		ctx context.Context, stakerPk string, paginationToken string,
	) (*DbResultMap[model.DelegationDocument], error)
}
