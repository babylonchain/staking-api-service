package db

import (
	"context"

	"github.com/babylonchain/staking-api-service/internal/db/model"
)

func (db *Database) SaveUnprocessableMessage(ctx context.Context, messageBody, receipt string) error {
	unprocessableMsgClient := db.Client.Database(db.DbName).Collection(model.UnprocessableMsgCollection)

	_, err := unprocessableMsgClient.InsertOne(ctx, model.NewUnprocessableMessageDocument(messageBody, receipt))
	if err != nil {
		return err
	}

	return nil
}
