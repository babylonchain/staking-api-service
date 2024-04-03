package model

import (
	"encoding/base64"
	"encoding/json"

	"github.com/babylonchain/staking-api-service/internal/types"
)

const DelegationCollection = "delegations"

type DelegationDocument struct {
	StakingTxHashHex      string                `bson:"_id"` // Primary key
	StakerPkHex           string                `bson:"staker_pk_hex"`
	FinalityProviderPkHex string                `bson:"finality_provider_pk_hex"`
	StakingValue          uint64                `bson:"staking_value"`
	StakingStartHeight    uint64                `bson:"staking_start_height"`
	StakingTimeLock       uint64                `bson:"staking_timelock"`
	StakingOutputIndex    uint64                `bson:"staking_output_index"`
	State                 types.DelegationState `bson:"state"`
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
		StakingStartHeight: d.StakingStartHeight,
	}
	token, err := page.GetPaginationToken()
	if err != nil {
		return "", err
	}
	return token, nil
}
