package services

type finalityProviderDescription struct {
	Moniker         string `json:"moniker"`
	Identity        string `json:"identity"`
	Website         string `json:"website"`
	SecurityContact string `json:"security_contact"`
	Details         string `json:"details"`
}

type FinalityProviderDetails struct {
	Description finalityProviderDescription `json:"description"`
	Commission  string                      `json:"commission"`
	BtcPk       string                      `json:"btc_pk"`
}

type GlobalParamsPublic struct {
	Tag               string                    `json:"tag"`
	CovenantPks       []string                  `json:"covenant_pks"`
	FinalityProviders []FinalityProviderDetails `json:"finality_providers"`
	CovenantQuoRum    uint64                    `json:"covenant_quorum"`
	UnbondingTime     uint64                    `json:"unbonding_time"`
	MaxStakingAmount  uint64                    `json:"max_staking_amount"`
	MinStakingAmount  uint64                    `json:"min_staking_amount"`
	MaxStakingTime    uint64                    `json:"max_staking_time"`
	MinStakingTime    uint64                    `json:"min_staking_time"`
}

func (s *Services) GetGlobalParams() *GlobalParamsPublic {
	var fpDetails []FinalityProviderDetails
	for _, finalityProvider := range s.params.FinalityProviders {
		description := finalityProviderDescription{
			Moniker:         finalityProvider.Description.Moniker,
			Identity:        finalityProvider.Description.Identity,
			Website:         finalityProvider.Description.Website,
			SecurityContact: finalityProvider.Description.SecurityContact,
			Details:         finalityProvider.Description.Details,
		}
		fpDetails = append(fpDetails, FinalityProviderDetails{
			Description: description,
			Commission:  finalityProvider.Commission,
			BtcPk:       finalityProvider.BtcPk,
		})
	}

	return &GlobalParamsPublic{
		Tag:               s.params.Tag,
		CovenantPks:       s.params.CovenantPks,
		FinalityProviders: fpDetails,
		CovenantQuoRum:    s.params.CovenantQuoRum,
		UnbondingTime:     s.params.UnbondingTime,
		MaxStakingAmount:  s.params.MaxStakingAmount,
		MinStakingAmount:  s.params.MinStakingAmount,
		MaxStakingTime:    s.params.MaxStakingTime,
		MinStakingTime:    s.params.MinStakingTime,
	}
}

// GetFinalityProvidersFromGlobalParams returns the finality providers from the global params.
// Those FP are treated as "active" finality providers.
func (s *Services) GetFinalityProvidersFromGlobalParams() []FinalityProviderDetails {
	var fpDetails []FinalityProviderDetails
	for _, finalityProvider := range s.params.FinalityProviders {
		description := finalityProviderDescription{
			Moniker:         finalityProvider.Description.Moniker,
			Identity:        finalityProvider.Description.Identity,
			Website:         finalityProvider.Description.Website,
			SecurityContact: finalityProvider.Description.SecurityContact,
			Details:         finalityProvider.Description.Details,
		}
		fpDetails = append(fpDetails, FinalityProviderDetails{
			Description: description,
			Commission:  finalityProvider.Commission,
			BtcPk:       finalityProvider.BtcPk,
		})
	}
	return fpDetails
}