package services

type GlobalParamsPublic struct {
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

func (s *Services) GetGlobalParamsPublic() *GlobalParamsPublic {
	return &GlobalParamsPublic{
		Tag:              s.params.Tag,
		CovenantPks:      s.params.CovenantPks,
		CovenantQuorum:   s.params.CovenantQuorum,
		UnbondingTime:    s.params.UnbondingTime,
		UnbondingFee:     s.params.UnbondingFee,
		MaxStakingAmount: s.params.MaxStakingAmount,
		MinStakingAmount: s.params.MinStakingAmount,
		MaxStakingTime:   s.params.MaxStakingTime,
		MinStakingTime:   s.params.MinStakingTime,
	}
}
