package types

import (
	"encoding/json"
	"os"
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

type FinalityProviders struct {
	FinalityProviders []FinalityProviderDetails `json:"finality_providers"`
}

func NewFinalityProviders(filePath string) ([]FinalityProviderDetails, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var finalityProviders FinalityProviders
	err = json.Unmarshal(data, &finalityProviders)
	if err != nil {
		return nil, err
	}

	return finalityProviders.FinalityProviders, nil
}
