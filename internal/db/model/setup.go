package model

import (
	"context"
	"fmt"
	"time"

	"github.com/babylonchain/staking-api-service/internal/config"
	"github.com/rs/zerolog/log"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	StatsLockCollection             = "stats_lock"
	OverallStatsCollection          = "overall_stats"
	FinalityProviderStatsCollection = "finality_providers_stats"
	StakerStatsCollection           = "staker_stats"
	DelegationCollection            = "delegations"
	TimeLockCollection              = "timelock_queue"
	UnbondingCollection             = "unbonding_queue"
	BtcInfoCollection               = "btc_info"
	UnprocessableMsgCollection      = "unprocessable_messages"
)

type index struct {
	Indexes map[string]int
	Unique  bool
}

var collections = map[string][]index{
	StatsLockCollection:             {{Indexes: map[string]int{}}},
	OverallStatsCollection:          {{Indexes: map[string]int{}}},
	FinalityProviderStatsCollection: {{Indexes: map[string]int{"active_tvl": -1}, Unique: false}},
	StakerStatsCollection:           {{Indexes: map[string]int{"active_tvl": -1}, Unique: false}},
	DelegationCollection: {
		{Indexes: map[string]int{"staker_pk_hex": 1, "staking_tx.start_height": -1}, Unique: false},
		{Indexes: map[string]int{"staker_btc_address.taproot_address": 1}, Unique: false},
	},
	TimeLockCollection:         {{Indexes: map[string]int{"expire_height": 1}, Unique: false}},
	UnbondingCollection:        {{Indexes: map[string]int{"unbonding_tx_hash_hex": 1}, Unique: true}},
	UnprocessableMsgCollection: {{Indexes: map[string]int{}}},
	BtcInfoCollection:          {{Indexes: map[string]int{}}},
}

func Setup(ctx context.Context, cfg *config.Config) error {
	clientOps := options.Client().ApplyURI(cfg.Db.Address)
	client, err := mongo.Connect(ctx, clientOps)
	if err != nil {
		return err
	}

	// Create a context with timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Access a database and create collections.
	database := client.Database(cfg.Db.DbName)

	// Create collections.
	for collection := range collections {
		createCollection(ctx, database, collection)
	}

	for name, idxs := range collections {
		for _, idx := range idxs {
			createIndex(ctx, database, name, idx)
		}
	}

	log.Info().Msg("Collections and Indexes created successfully.")
	return nil
}

func createCollection(ctx context.Context, database *mongo.Database, collectionName string) {
	// Check if the collection already exists.
	if _, err := database.Collection(collectionName).Indexes().CreateOne(ctx, mongo.IndexModel{}); err != nil {
		log.Debug().Msg(fmt.Sprintf("Collection maybe already exists: %s, skip the rest. info: %s", collectionName, err))
		return
	}

	// Create the collection.
	if err := database.CreateCollection(ctx, collectionName); err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("Failed to create collection: " + collectionName)
		return
	}

	log.Debug().Msg("Collection created successfully: " + collectionName)
}

func createIndex(ctx context.Context, database *mongo.Database, collectionName string, idx index) {
	if len(idx.Indexes) == 0 {
		return
	}

	indexKeys := bson.D{}
	for k, v := range idx.Indexes {
		indexKeys = append(indexKeys, bson.E{Key: k, Value: v})
	}

	index := mongo.IndexModel{
		Keys:    indexKeys,
		Options: options.Index().SetUnique(idx.Unique),
	}

	if _, err := database.Collection(collectionName).Indexes().CreateOne(ctx, index); err != nil {
		log.Debug().Msg(fmt.Sprintf("Failed to create index on collection '%s': %v", collectionName, err))
		return
	}

	log.Debug().Msg("Index created successfully on collection: " + collectionName)
}
