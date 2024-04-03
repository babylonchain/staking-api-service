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
		// Find the existing delegation document first, it will be used later in the transaction
		delegationFilter := bson.M{
			"_id":   stakingTxHashHex,
			"state": types.Active,
		}
		var delegationDocument model.DelegationDocument
		err = delegationClient.FindOne(sessCtx, delegationFilter).Decode(&delegationDocument)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				return nil, &NotFoundError{
					Key:     stakingTxHashHex,
					Message: "no active delegation found for unbonding request",
				}
			}
			return nil, err
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
				Message: "delegation not found or not eligible for unbonding",
			}
		}

		// Insert the unbonding transaction document
		unbondingDocument := model.UnbondingDocument{
			StakerPkHex:        delegationDocument.StakerPkHex,
			FinalityPkHex:      delegationDocument.FinalityProviderPkHex,
			UnbondingTxSigHex:  signatureHex,
			State:              model.UnbondingInitialState,
			UnbondingTxHashHex: txHashHex,
			UnbondingTxHex:     txHex,
			StakingTxHex:       delegationDocument.StakingTx.TxHex,
			StakingOutputIndex: delegationDocument.StakingTx.OutputIndex,
			StakingTimelock:    delegationDocument.StakingTx.TimeLock,
			StakingTxHashHex:   stakingTxHashHex,
			StakingAmount:      delegationDocument.StakingValue,
		}
		_, err = unbondingClient.InsertOne(sessCtx, unbondingDocument)
		if err != nil {
			var writeErr mongo.WriteException
			if errors.As(err, &writeErr) {
				for _, e := range writeErr.WriteErrors {
					if mongo.IsDuplicateKeyError(e) {
						return nil, &DuplicateKeyError{
							Key:     txHashHex,
							Message: "unbonding transaction already exists",
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
