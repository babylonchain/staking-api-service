package db

import (
	"context"

	"github.com/babylonchain/staking-api-service/internal/config"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Database struct {
	DbName string
	Client *mongo.Client
	cfg    config.DbConfig
}

type DbResultMap[T any] struct {
	Data            []T    `json:"data"`
	PaginationToken string `json:"paginationToken"`
}

func New(ctx context.Context, cfg config.DbConfig) (*Database, error) {
	clientOps := options.Client().ApplyURI(cfg.Address)
	client, err := mongo.Connect(ctx, clientOps)
	if err != nil {
		return nil, err
	}

	return &Database{
		DbName: cfg.DbName,
		Client: client,
		cfg:    cfg,
	}, nil
}

func (db *Database) Ping(ctx context.Context) error {
	err := db.Client.Ping(ctx, nil)
	if err != nil {
		return err
	}
	return nil
}

// This function is used to build the result map with pagination token
// It will return the result map with pagination token if the result length is equal to the fetch limit
// Otherwise it will return the result map without pagination token. i.e pagination token will be empty string
func toResultMapWithPaginationToken[T any](cfg config.DbConfig, result []T, paginationKeyBuilder func(T) (string, error)) (*DbResultMap[T], error) {
	if len(result) > 0 && len(result) == int(cfg.MaxPaginationLimit) {
		paginationToken, err := paginationKeyBuilder(result[len(result)-1])
		if err != nil {
			return nil, err
		}
		return &DbResultMap[T]{
			Data:            result,
			PaginationToken: paginationToken,
		}, nil

	}

	return &DbResultMap[T]{
		Data:            result,
		PaginationToken: "",
	}, nil
}
