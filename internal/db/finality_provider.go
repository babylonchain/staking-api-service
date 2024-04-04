package db

import (
	"context"

	"github.com/babylonchain/staking-api-service/internal/db/model"
	"go.mongodb.org/mongo-driver/bson"
)

// FindFinalityProvidersByPkHex fetches finality providers by their primary key hex.
// It returns a map of finality providers with their BTC Pk hex as the key.
func (db *Database) FindFinalityProvidersByPkHex(ctx context.Context, pkHex []string) (map[string]model.FinalityProviderDocument, error) {
	client := db.Client.Database(db.DbName).Collection(model.FinalityProviderCollection)
	finalityProvidersMap := make(map[string]model.FinalityProviderDocument)

	batchSize := int(db.cfg.DbBatchSizeLimist)
	for i := 0; i < len(pkHex); i += batchSize {
		end := i + batchSize
		if end > len(pkHex) {
			end = len(pkHex)
		}
		batch := pkHex[i:end]

		filter := bson.M{"_id": bson.M{"$in": batch}}
		cursor, err := client.Find(ctx, filter)
		if err != nil {
			return nil, err
		}

		var finalityProviders []model.FinalityProviderDocument
		if err = cursor.All(ctx, &finalityProviders); err != nil {
			cursor.Close(ctx)
			return nil, err
		}
		cursor.Close(ctx)

		for _, fp := range finalityProviders {
			finalityProvidersMap[fp.FinalityProviderPkHex] = fp
		}
	}

	return finalityProvidersMap, nil
}
