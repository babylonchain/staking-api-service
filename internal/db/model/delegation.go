package model

import (
	"encoding/base64"
	"encoding/json"

	"github.com/babylonchain/staking-api-service/internal/types"
)

const DelegationCollection = "delegations"

type TimelockTransaction struct {
	TxHex          string `bson:"tx_hex"`
	OutputIndex    uint64 `bson:"output_index"`
	StartTimestamp string `bson:"start_timestamp"`
	StartHeight    uint64 `bson:"start_height"`
	TimeLock       uint64 `bson:"timelock"`
}

type DelegationDocument struct {
	StakingTxHashHex      string                `bson:"_id"` // Primary key
	StakingValue          uint64                `bson:"staking_value"`
	State                 types.DelegationState `bson:"state"`
	StakerPkHex           string                `bson:"staker_pk_hex"`
	FinalityProviderPkHex string                `bson:"finality_provider_pk_hex"`
	StakingTx             *TimelockTransaction  `bson:"staking_tx"` // Always exist
	UnbondingTx           *TimelockTransaction  `bson:"unbonding_tx,omitempty"`
}

type DelegationByStakerPagination struct {
	StakingTxHashHex   string `json:"staking_tx_hash_hex"`
	StakingStartHeight uint64 `json:"staking_start_height"`
}

func DecodeDelegationByStakerPaginationToken(token string) (*DelegationByStakerPagination, error) {
	tokenBytes, err := base64.URLEncoding.DecodeString(token)
	if err != nil {
		return nil, err
	}
	var d DelegationByStakerPagination
	err = json.Unmarshal(tokenBytes, &d)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (d *DelegationByStakerPagination) GetPaginationToken() (string, error) {
	tokenBytes, err := json.Marshal(d)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(tokenBytes), nil
}

func BuildDelegationByStakerPaginationToken(d DelegationDocument) (string, error) {
	page := &DelegationByStakerPagination{
		StakingTxHashHex:   d.StakingTxHashHex,
		StakingStartHeight: d.StakingTx.StartHeight,
	}
	token, err := page.GetPaginationToken()
	if err != nil {
		return "", err
	}
	return token, nil
}
