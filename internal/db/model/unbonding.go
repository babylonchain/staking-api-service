package model

const (
	UnbondingCollection   = "unbonding_queue"
	UnbondingInitialState = "INSERTED"
)

type UnbondingDocument struct {
	ID                       string `bson:"_id"`                   // Primary key
	UnbondingTxHashHex       string `bson:"unbonding_tx_hash_hex"` // Unique Index
	UnbondingTxHex           string `bson:"unbonding_tx_hex"`
	StakerSignedSignatureHex string `bson:"staker_signed_signature_hex"`
	State                    string `bson:"state"`
}
