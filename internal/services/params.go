package services

import (
	"github.com/babylonchain/staking-api-service/internal/types"
)

type VersionedGlobalParamsPublic struct {
	Version          uint64   `json:"version"`
	ActivationHeight uint64   `json:"activation_height"`
	StakingCap       uint64   `json:"staking_cap"`
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

type GlobalParamsPublic struct {
	Versions []VersionedGlobalParamsPublic `json:"versions"`
}

func (s *Services) GetGlobalParamsPublic() *GlobalParamsPublic {
	var versionedParams []VersionedGlobalParamsPublic
	for _, version := range s.params.Versions {
		versionedParams = append(versionedParams, VersionedGlobalParamsPublic{
			Version:          version.Version,
			ActivationHeight: version.ActivationHeight,
			StakingCap:       version.StakingCap,
			Tag:              version.Tag,
			CovenantPks:      version.CovenantPks,
			CovenantQuorum:   version.CovenantQuorum,
			UnbondingTime:    version.UnbondingTime,
			UnbondingFee:     version.UnbondingFee,
			MaxStakingAmount: version.MaxStakingAmount,
			MinStakingAmount: version.MinStakingAmount,
			MaxStakingTime:   version.MaxStakingTime,
			MinStakingTime:   version.MinStakingTime,
		})
	}
	return &GlobalParamsPublic{
		Versions: versionedParams,
	}
}

func (s *Services) GetVersionedGlobalParamsByHeight(height uint64) *types.VersionedGlobalParams {
	// Find the version by height
	for _, version := range s.params.Versions {
		if version.ActivationHeight <= height {
			return version
		}
	}

	return nil
}
