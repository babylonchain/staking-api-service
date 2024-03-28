package model

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const TimeLockCollection = "timelock_queue"

type TimeLockDocument struct {
	ID               primitive.ObjectID `bson:"_id"`
	StakingTxHashHex string             `bson:"_id"`
	ExpireHeight     uint64             `bson:"expire_height"`
}
