package db

import (
	"context"
	"errors"

	"github.com/babylonchain/staking-api-service/internal/db/model"
	"github.com/babylonchain/staking-api-service/internal/types"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func (db *Database) SaveUnbondingTx(
	ctx context.Context, stakingTxHashHex, txHashHex, txHex, signatureHex string,
) error {
	delegationClient := db.Client.Database(db.DbName).Collection(model.DelegationCollection)
	unbondingClient := db.Client.Database(db.DbName).Collection(model.UnbondingCollection)

	// Start a session
	session, err := db.Client.StartSession()
	if err != nil {
		return err
	}
	defer session.EndSession(ctx)

	// Define the work to be done in the transaction
	transactionWork := func(sessCtx mongo.SessionContext) (interface{}, error) {
		// Handle the delegation update
		delegationFilter := bson.M{
			"_id":   stakingTxHashHex,
			"state": types.Active,
		}
		// Update the state to UnbondingRequested
		delegationUpdate := bson.M{"$set": bson.M{"state": types.UnbondingRequested}}
		result, err := delegationClient.UpdateOne(sessCtx, delegationFilter, delegationUpdate)
		if err != nil {
			return nil, err
		}

		if result.MatchedCount == 0 {
			return nil, &NotFoundError{
				Key:     stakingTxHashHex,
				Message: "no active delegation found during state update for unbonding",
			}
		}

		// Insert the unbonding transaction document
		unbondingDocument := model.UnbondingDocument{
			UnbondingTxHashHex:       txHashHex,
			UnbondingTxHex:           txHex,
			StakerSignedSignatureHex: signatureHex,
			State:                    model.UnbondingInitialState,
		}
		_, err = unbondingClient.InsertOne(sessCtx, unbondingDocument)
		if err != nil {
			var writeErr mongo.WriteException
			if errors.As(err, &writeErr) {
				for _, e := range writeErr.WriteErrors {
					if mongo.IsDuplicateKeyError(e) {
						return nil, &DuplicateKeyError{
							Key:     txHashHex,
							Message: "unbonding tx hash hex already exists in collection",
						}
					}
				}
			}
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
