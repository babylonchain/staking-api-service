package db

import (
	"context"

	"github.com/babylonchain/staking-api-service/internal/db/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func (db *Database) SaveUnprocessableMessage(ctx context.Context, messageBody, receipt string) error {
	unprocessableMsgClient := db.Client.Database(db.DbName).Collection(model.UnprocessableMsgCollection)

	_, err := unprocessableMsgClient.InsertOne(ctx, model.NewUnprocessableMessageDocument(messageBody, receipt))
	if err != nil {
		return err
	}

	return nil
}

func (db *Database) GetUnprocessableMessages(ctx context.Context) (*mongo.Cursor, error) {
	unprocessableMsgClient := db.Client.Database(db.DbName).Collection(model.UnprocessableMsgCollection)
	filter := bson.M{}
	return unprocessableMsgClient.Find(ctx, filter)
}

func (db *Database) DeleteUnprocessableMessage(ctx context.Context, Receipt interface{}) error {
	unprocessableMsgClient := db.Client.Database(db.DbName).Collection(model.UnprocessableMsgCollection)
	filter := bson.M{"receipt": Receipt}
	_, err := unprocessableMsgClient.DeleteOne(ctx, filter)
	return err
}
