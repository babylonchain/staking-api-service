package types

import (
	"encoding/json"
	"os"
)

type GlobalParams struct {
	Tag              string   `json:"tag"`
	CovenantPks      []string `json:"covenant_pks"`
	CovenantQuorum   uint64   `json:"covenant_quorum"`
	UnbondingTime    uint64   `json:"unbonding_time"`
	UnbondingFee     uint64   `json:"unbonding_fee"`
	MaxStakingAmount uint64   `json:"max_staking_amount"`
	MinStakingAmount uint64   `json:"min_staking_amount"`
	MaxStakingTime   uint64   `json:"max_staking_time"`
	MinStakingTime   uint64   `json:"min_staking_time"`
}

func NewGlobalParams(filePath string) (*GlobalParams, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var globalParams GlobalParams
	err = json.Unmarshal(data, &globalParams)
	if err != nil {
		return nil, err
	}

	return &globalParams, nil
}
