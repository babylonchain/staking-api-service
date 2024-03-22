package db

import (
	"context"
	"errors"

	"github.com/babylonchain/staking-api-service/internal/db/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// TODO: This will be dynamic from the client with max limit from config
const FetchLimit = 10

type Database struct {
	DbName string
	Client *mongo.Client
}

type DbResultMap[T any] struct {
	Data            []T    `json:"data"`
	PaginationToken string `json:"paginationToken"`
}

func New(ctx context.Context, dbName string, dbURI string) (*Database, error) {
	clientOps := options.Client().ApplyURI(dbURI)
	client, err := mongo.Connect(ctx, clientOps)
	if err != nil {
		return nil, err
	}

	return &Database{
		DbName: dbName,
		Client: client,
	}, nil
}

func (db *Database) Ping(ctx context.Context) error {
	err := db.Client.Ping(ctx, nil)
	if err != nil {
		return err
	}
	return nil
}

func (db *Database) SaveActiveStakingDelegation(
	ctx context.Context, stakingTxHashHex, stakerPhHex,
	finalityProviderPkHex string, amount, startHeight, timelock uint64,
) error {
	client := db.Client.Database(db.DbName).Collection(model.DelegationCollection)
	document := model.DelegationDocument{
		StakingTxHex:          stakingTxHashHex, // Primary key of db collection
		StakerPkHex:           stakerPhHex,
		FinalityProviderPkHex: finalityProviderPkHex,
		StakingValue:          amount,
		StakingStartkHeight:   startHeight,
		StakingTimeLock:       timelock,
		State:                 model.Active,
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
						Message: "Staking transaction already exists in the database",
					}
				}
			}
		}
		return err
	}
	return nil
}

func (db *Database) FindDelegationsByStakerPk(ctx context.Context, stakerPk string, paginationToken string) (DbResultMap[model.DelegationDocument], error) {
	var resultMap DbResultMap[model.DelegationDocument]
	client := db.Client.Database(db.DbName).Collection(model.DelegationCollection)

	filter := bson.M{"staker_pk_hex": stakerPk}
	options := options.Find().SetSort(bson.M{"staking_start_height": -1}) // Sorting in descending order

	options.SetLimit(FetchLimit)
	// Decode the pagination token first if it exist
	if paginationToken != "" {
		decodedToken, err := model.DecodeDelegationByStakerPaginationToken(paginationToken)
		if err != nil {
			return DbResultMap[model.DelegationDocument]{}, &InvalidPaginationTokenError{
				Message: "Invalid pagination token",
			}
		}
		lastSeenHeight := decodedToken.StakingStartkHeight
		filter["staking_start_height"] = bson.M{"$lt": lastSeenHeight}
	}

	cursor, err := client.Find(ctx, filter, options)
	if err != nil {
		return resultMap, err
	}
	defer cursor.Close(ctx)

	var delegations []model.DelegationDocument
	if err = cursor.All(ctx, &delegations); err != nil {
		return resultMap, err
	}

	return toResultMapWithPaginationToken(delegations, model.BuildDelegationByStakerPaginationToken)
}

func toResultMapWithPaginationToken[T any](result []T, paginationKeyBuilder func(T) (string, error)) (DbResultMap[T], error) {
	if len(result) > 0 && len(result) == FetchLimit {
		paginationToken, err := paginationKeyBuilder(result[len(result)-1])
		if err != nil {
			return DbResultMap[T]{}, err
		}
		return DbResultMap[T]{
			Data:            result,
			PaginationToken: paginationToken,
		}, nil

	}

	return DbResultMap[T]{
		Data:            result,
		PaginationToken: "",
	}, nil
}
