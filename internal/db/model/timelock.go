package model

const TimeLockCollection = "timelock_queue"

type TimeLockDocument struct {
	StakingTxHashHex string `bson:"staking_tx_hash_hex"`
	ExpireHeight     uint64 `bson:"expire_height"`
}

func NewTimeLockDocument(stakingTxHashHex string, expireHeight uint64) *TimeLockDocument {
	return &TimeLockDocument{
		StakingTxHashHex: stakingTxHashHex,
		ExpireHeight:     expireHeight,
	}
}
