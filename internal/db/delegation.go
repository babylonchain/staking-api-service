package db

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/babylonchain/staking-api-service/internal/db/model"
	"github.com/babylonchain/staking-api-service/internal/types"
)

func (db *Database) SaveActiveStakingDelegation(
	ctx context.Context, stakingTxHashHex, stakerPhHex,
	finalityProviderPkHex string, amount, startHeight, timelock, stakingOutputIndex uint64,
) error {
	client := db.Client.Database(db.DbName).Collection(model.DelegationCollection)
	document := model.DelegationDocument{
		StakingTxHashHex:      stakingTxHashHex, // Primary key of db collection
		StakerPkHex:           stakerPhHex,
		FinalityProviderPkHex: finalityProviderPkHex,
		StakingValue:          amount,
		StakingStartHeight:    startHeight,
		StakingTimeLock:       timelock,
		StakingOutputIndex:    stakingOutputIndex,
		State:                 types.Active,
	}
	_, err := client.InsertOne(ctx, document)
	if err != nil {
		var writeErr mongo.WriteException
		if errors.As(err, &writeErr) {
			for _, e := range writeErr.WriteErrors {
				if mongo.IsDuplicateKeyError(e) {
					// Return the custom error type so that we can return 4xx errors to client
					return &DuplicateKeyError{
						Key:     stakingTxHashHex,
						Message: "Delegation already exists",
					}
				}
			}
		}
		return err
	}
	return nil
}

func (db *Database) FindDelegationsByStakerPk(ctx context.Context, stakerPk string, paginationToken string) (*DbResultMap[model.DelegationDocument], error) {
	client := db.Client.Database(db.DbName).Collection(model.DelegationCollection)

	filter := bson.M{"staker_pk_hex": stakerPk}
	options := options.Find().SetSort(bson.M{"staking_start_height": -1}) // Sorting in descending order

	options.SetLimit(FetchLimit)
	// Decode the pagination token first if it exist
	if paginationToken != "" {
		decodedToken, err := model.DecodeDelegationByStakerPaginationToken(paginationToken)
		if err != nil {
			return nil, &InvalidPaginationTokenError{
				Message: "Invalid pagination token",
			}
		}
		filter = bson.M{
			"$or": []bson.M{
				{"staking_start_height": bson.M{"$lt": decodedToken.StakingStartHeight}},
				{"staking_start_height": decodedToken.StakingStartHeight, "_id": bson.M{"$gt": decodedToken.StakingTxHashHex}},
			},
		}
	}

	cursor, err := client.Find(ctx, filter, options)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var delegations []model.DelegationDocument
	if err = cursor.All(ctx, &delegations); err != nil {
		return nil, err
	}

	return toResultMapWithPaginationToken(delegations, model.BuildDelegationByStakerPaginationToken)
}

func (db *Database) FindDelegationByTxHashHex(ctx context.Context, stakingTxHashHex string) (*model.DelegationDocument, error) {
	client := db.Client.Database(db.DbName).Collection(model.DelegationCollection)
	filter := bson.M{"_id": stakingTxHashHex}
	var delegation model.DelegationDocument
	err := client.FindOne(ctx, filter).Decode(&delegation)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, &NotFoundError{
				Key:     stakingTxHashHex,
				Message: "Delegation not found",
			}
		}
		return nil, err
	}
	return &delegation, nil
}
