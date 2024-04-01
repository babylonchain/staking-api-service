package db

import (
	"context"

	"github.com/babylonchain/staking-api-service/internal/db/model"
)

func (db *Database) SaveTimeLockExpireCheck(
	ctx context.Context, stakingTxHashHex string,
	expireHeight uint64, txType string,
) error {
	client := db.Client.Database(db.DbName).Collection(model.TimeLockCollection)
	document := model.NewTimeLockDocument(stakingTxHashHex, expireHeight, txType)
	_, err := client.InsertOne(ctx, document)
	if err != nil {
		return err
	}
	return nil
}
