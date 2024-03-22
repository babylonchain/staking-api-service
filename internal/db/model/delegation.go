package model

import (
	"encoding/base64"
	"encoding/json"
)

const DelegationCollection = "delegations"

type DelegationState string

const (
	Active             DelegationState = "active"
	UnbondingRequested DelegationState = "unbonding_requested"
	Unbonding          DelegationState = "unbonding"
	Unbonded           DelegationState = "unbonded"
	Withdrawn          DelegationState = "withdrawn"
)

func (s DelegationState) ToString() string {
	return string(s)
}

type DelegationDocument struct {
	StakingTxHex          string          `bson:"_id"` // Primary key
	StakerPkHex           string          `bson:"staker_pk_hex"`
	FinalityProviderPkHex string          `bson:"finality_provider_pk_hex"`
	StakingValue          uint64          `bson:"staking_value"`
	StakingStartkHeight   uint64          `bson:"staking_start_height"`
	StakingTimeLock       uint64          `bson:"staking_timelock"`
	State                 DelegationState `bson:"state"`
}

type DelegationByStakerPagination struct {
	StakingStartkHeight uint64 `json:"staking_start_height"`
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
		StakingStartkHeight: d.StakingStartkHeight,
	}
	token, err := page.GetPaginationToken()
	if err != nil {
		return "", err
	}
	return token, nil
}
