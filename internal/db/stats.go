package db

import (
	"context"
	"math/rand"

	"github.com/babylonchain/staking-api-service/internal/db/model"
	"github.com/babylonchain/staking-api-service/internal/types"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (db *Database) GetOrCreateStatsLock(
	ctx context.Context, stakingTxHashHex string, txType string,
) (*model.StatsLockDocument, error) {
	client := db.Client.Database(db.DbName).Collection(model.StatsLockCollection)
	id := constructStatsLockId(stakingTxHashHex, txType)
	filter := bson.M{"_id": id}
	// Define the default document to be inserted if not found
	// This setOnInsert will only be applied if the document is not found
	update := bson.M{
		"$setOnInsert": model.NewStatsLockDocument(
			id,
			false,
			false,
			false,
		),
	}
	opts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)

	var result model.StatsLockDocument
	err := client.FindOneAndUpdate(ctx, filter, update, opts).Decode(&result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// IncrementOverallStats increments the overall stats for the given staking tx hash.
// This method is idempotent, only the first call will be processed. Otherwise it will return a notFoundError for duplicates
func (db *Database) IncrementOverallStats(
	ctx context.Context, stakingTxHashHex string, amount int64,
) error {
	return db.updateOverallStats(ctx, types.ActiveTxType.ToString(), stakingTxHashHex, amount)
}

// SubtractOverallStats decrements the overall stats for the given staking tx hash
// This method is idempotent, only the first call will be processed. Otherwise it will return a notFoundError for duplicates
func (db *Database) SubtractOverallStats(
	ctx context.Context, stakingTxHashHex string, amount int64,
) error {
	if amount > 0 {
		return &InvalidArgumentError{
			Key:     "amount",
			Message: "amount should be negative for unbonding operation",
		}

	}
	return db.updateOverallStats(ctx, types.UnbondingTxType.ToString(), stakingTxHashHex, amount)
}

// Genrate the id for the overall stats document. Id is a random number ranged from 0-LogicalShardCount
// It's a logical shard to avoid locking the same field during concurrent writes
// The sharding number should never be reduced after roll out
func (db *Database) generateOverallStatsId() uint64 {
	return uint64(rand.Intn(int(db.cfg.LogicalShardCount)))
}

func (db *Database) updateOverallStats(ctx context.Context, txType, stakingTxHashHex string, amount int64) error {
	overallStatsClient := db.Client.Database(db.DbName).Collection(model.OverallStatsCollection)

	// Start a session
	session, err := db.Client.StartSession()
	if err != nil {
		return err
	}
	defer session.EndSession(ctx)

	// Define the work to be done in the transaction
	transactionWork := func(sessCtx mongo.SessionContext) (interface{}, error) {
		err := db.updateStatsLockByFieldName(sessCtx, stakingTxHashHex, txType, "overall_stats")
		if err != nil {
			return nil, err
		}

		upsertFilter := bson.M{"_id": db.generateOverallStatsId()}

		var upsertUpdate bson.M
		// If the amount is negative, it means we are unbonding.
		// Only the active tvl and delegation numbers should be decremented
		if amount < 0 {
			upsertUpdate = bson.M{
				"$inc": bson.M{
					"active_tvl":         amount,
					"active_delegations": -1,
				},
			}
		} else {
			upsertUpdate = bson.M{
				"$inc": bson.M{
					"active_tvl":         amount,
					"total_tvl":          amount,
					"active_delegations": 1,
					"total_delegations":  1,
				},
			}
		}
		_, err = overallStatsClient.UpdateOne(sessCtx, upsertFilter, upsertUpdate, options.Update().SetUpsert(true))
		if err != nil {
			return nil, err
		}
		return nil, nil
	}

	// Execute the transaction
	_, err = session.WithTransaction(ctx, transactionWork)
	if err != nil {
		return err
	}

	return nil
}

func (db *Database) updateStatsLockByFieldName(ctx context.Context, stakingTxHashHex, txType string, fieldName string) error {
	statsLockClient := db.Client.Database(db.DbName).Collection(model.StatsLockCollection)
	filter := bson.M{"_id": constructStatsLockId(stakingTxHashHex, txType), fieldName: false}
	update := bson.M{"$set": bson.M{fieldName: true}}
	result, err := statsLockClient.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return &NotFoundError{
			Key:     stakingTxHashHex,
			Message: "document already processed or does not exist",
		}
	}
	return nil
}

func constructStatsLockId(stakingTxHashHex, txType string) string {
	return stakingTxHashHex + ":" + txType
}
