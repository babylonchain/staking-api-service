package types

import (
	"encoding/json"
	"os"

	"github.com/btcsuite/btcd/btcec/v2"

	"github.com/babylonchain/staking-api-service/internal/utils"
)

type FinalityProviderDescription struct {
	Moniker         string `json:"moniker"`
	Identity        string `json:"identity"`
	Website         string `json:"website"`
	SecurityContact string `json:"security_contact"`
	Details         string `json:"details"`
}

type FinalityProviderDetails struct {
	Description FinalityProviderDescription `json:"description"`
	Commission  string                      `json:"commission"`
	BtcPk       string                      `json:"btc_pk"`
}

type GlobalParams struct {
	Tag               string
	CovenantPks       []*btcec.PublicKey
	FinalityProviders []FinalityProviderDetails
	CovenantQuorum    uint64
	UnbondingTime     uint64
	MaxStakingAmount  uint64
	MinStakingAmount  uint64
	MaxStakingTime    uint64
	MinStakingTime    uint64
}

type internalGlobalParams struct {
	Tag               string                    `json:"tag"`
	CovenantPks       []string                  `json:"covenant_pks"`
	FinalityProviders []FinalityProviderDetails `json:"finality_providers"`
	CovenantQuorum    uint64                    `json:"covenant_quorum"`
	UnbondingTime     uint64                    `json:"unbonding_time"`
	MaxStakingAmount  uint64                    `json:"max_staking_amount"`
	MinStakingAmount  uint64                    `json:"min_staking_amount"`
	MaxStakingTime    uint64                    `json:"max_staking_time"`
	MinStakingTime    uint64                    `json:"min_staking_time"`
}

func NewGlobalParams(filePath string) (*GlobalParams, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var globalParams internalGlobalParams
	err = json.Unmarshal(data, &globalParams)
	if err != nil {
		return nil, err
	}

	covenantPks, err := utils.GetBtcPksFromStrings(globalParams.CovenantPks)
	if err != nil {
		return nil, err
	}

	return &GlobalParams{
		Tag:               globalParams.Tag,
		CovenantPks:       covenantPks,
		FinalityProviders: globalParams.FinalityProviders,
		CovenantQuorum:    globalParams.CovenantQuorum,
		UnbondingTime:     globalParams.UnbondingTime,
		MaxStakingAmount:  globalParams.MaxStakingAmount,
		MinStakingAmount:  globalParams.MinStakingAmount,
		MaxStakingTime:    globalParams.MaxStakingTime,
		MinStakingTime:    globalParams.MinStakingTime,
	}, nil
}
