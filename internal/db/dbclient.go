package db

import (
	"context"
	"time"

	"github.com/babylonchain/staking-api-service/internal/config"
	"github.com/babylonchain/staking-api-service/internal/observability/metrics"
	"github.com/rs/zerolog/log"
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
	client, err := connect(ctx, cfg.Address)
	if err != nil {
		return nil, err
	}

	return &Database{
		DbName: cfg.DbName,
		Client: client,
		cfg:    cfg,
	}, nil
}

func connect(ctx context.Context, addressUri string) (*mongo.Client, error) {
	clientOps := options.Client().ApplyURI(addressUri)
	client, err := mongo.Connect(ctx, clientOps)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func (db *Database) reconnect(ctx context.Context) error {
	client, err := connect(ctx, db.cfg.Address)
	if err != nil {
		return err
	}
	db.Client = client
	return nil
}

// StartConnectionCheckRoutine starts a routine to check the database connection
// If the ping fails, it will attempt to reconnect.
func (db *Database) StartConnectionCheckRoutine(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(db.cfg.ConnectionCheckPeriod)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := db.Ping(ctx); err != nil {
					log.Ctx(ctx).Error().Err(err).Msg("Failed to ping database, reconnecting!")
					metrics.RecordDbOperationFailure("ping")
					// Attempt to reconnect if ping fails
					db.reconnect(ctx)
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (db *Database) Ping(ctx context.Context) error {
	err := db.Client.Ping(ctx, nil)
	if err != nil {
		metrics.RecordDbOperationFailure("ping")
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
