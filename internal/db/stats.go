package db

import (
	"context"
	"math/rand"

	"github.com/babylonchain/staking-api-service/internal/db/model"
	"github.com/babylonchain/staking-api-service/internal/types"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
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
	ctx context.Context, stakingTxHashHex string, amount uint64,
) error {
	upsertUpdate := bson.M{
		"$inc": bson.M{
			"active_tvl":         int64(amount),
			"total_tvl":          int64(amount),
			"active_delegations": 1,
			"total_delegations":  1,
		},
	}
	return db.updateOverallStats(ctx, types.Active.ToString(), stakingTxHashHex, upsertUpdate)
}

// SubtractOverallStats decrements the overall stats for the given staking tx hash
// This method is idempotent, only the first call will be processed. Otherwise it will return a notFoundError for duplicates
func (db *Database) SubtractOverallStats(
	ctx context.Context, stakingTxHashHex string, amount uint64,
) error {
	upsertUpdate := bson.M{
		"$inc": bson.M{
			"active_tvl":         -int64(amount),
			"active_delegations": -1,
		},
	}
	return db.updateOverallStats(ctx, types.Unbonded.ToString(), stakingTxHashHex, upsertUpdate)
}

// Genrate the id for the overall stats document. Id is a random number ranged from 0-LogicalShardCount
// It's a logical shard to avoid locking the same field during concurrent writes
// The sharding number should never be reduced after roll out
func (db *Database) generateOverallStatsId() uint64 {
	return uint64(rand.Intn(int(db.cfg.LogicalShardCount)))
}

func (db *Database) updateOverallStats(ctx context.Context, state, stakingTxHashHex string, upsertUpdate primitive.M) error {
	overallStatsClient := db.Client.Database(db.DbName).Collection(model.OverallStatsCollection)

	// Start a session
	session, err := db.Client.StartSession()
	if err != nil {
		return err
	}
	defer session.EndSession(ctx)

	// Define the work to be done in the transaction
	transactionWork := func(sessCtx mongo.SessionContext) (interface{}, error) {
		err := db.updateStatsLockByFieldName(sessCtx, stakingTxHashHex, state, "overall_stats")
		if err != nil {
			return nil, err
		}

		upsertFilter := bson.M{"_id": db.generateOverallStatsId()}

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

func (db *Database) updateStatsLockByFieldName(ctx context.Context, stakingTxHashHex, state string, fieldName string) error {
	statsLockClient := db.Client.Database(db.DbName).Collection(model.StatsLockCollection)
	filter := bson.M{"_id": constructStatsLockId(stakingTxHashHex, state), fieldName: false}
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

func constructStatsLockId(stakingTxHashHex, state string) string {
	return stakingTxHashHex + ":" + state
}
