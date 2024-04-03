package model

const FinalityProviderCollection = "finality_providers"

type FinalityProviderDocument struct {
	FinalityProviderPkHex string `bson:"_id"`
	ActiveTvl             uint64 `bson:"active_tvl"`
	TotalTvl              uint64 `bson:"total_tvl"`
	ActiveDelegations     uint64 `bson:"active_delegations"`
	TotalDelegations      uint64 `bson:"total_delegations"`
}
