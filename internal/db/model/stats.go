package model

import (
	"encoding/base64"
	"encoding/json"
)

const (
	StatsLockCollection             = "stats_lock"
	OverallStatsCollection          = "overall_stats"
	FinalityProviderStatsCollection = "finality_providers_stats"
	StakerStatsCollection           = "staker_stats"
)

// StatsLockDocument represents the document in the stats lock collection
// It's used as a lock to prevent concurrent stats calculation for the same staking tx hash
// As well as to prevent the same staking tx hash + txType to be processed multiple times
// The already processed stats will be marked as true in the document
type StatsLockDocument struct {
	Id                    string `bson:"_id"`
	OverallStats          bool   `bson:"overall_stats"`
	StakerStats           bool   `bson:"staker_stats"`
	FinalityProviderStats bool   `bson:"finality_provider_stats"`
}

func NewStatsLockDocument(
	id string, overallStats, stakerStats, finalityProviderStats bool,
) *StatsLockDocument {
	return &StatsLockDocument{
		Id:                    id,
		OverallStats:          overallStats,
		StakerStats:           stakerStats,
		FinalityProviderStats: finalityProviderStats,
	}
}

type OverallStatsDocument struct {
	Id                string `bson:"_id"`
	ActiveTvl         int64  `bson:"active_tvl"`
	TotalTvl          int64  `bson:"total_tvl"`
	ActiveDelegations int64  `bson:"active_delegations"`
	TotalDelegations  int64  `bson:"total_delegations"`
	TotalStakers      uint64 `bson:"total_stakers"`
}

type FinalityProviderStatsDocument struct {
	Id                string `bson:"_id"` // FinalityProviderPkHex:shard-number
	ActiveTvl         int64  `bson:"active_tvl"`
	TotalTvl          int64  `bson:"total_tvl"`
	ActiveDelegations int64  `bson:"active_delegations"`
	TotalDelegations  int64  `bson:"total_delegations"`
}

type StakerStatsDocument struct {
	StakerPkHex       string `bson:"_id"`
	ActiveTvl         int64  `bson:"active_tvl"`
	TotalTvl          int64  `bson:"total_tvl"`
	ActiveDelegations int64  `bson:"active_delegations"`
	TotalDelegations  int64  `bson:"total_delegations"`
}

// StakerStatsByStakerPagination is used to paginate the top stakers by active tvl
// ActiveTvl is used as the sorting key, whereas StakerPkHex is used as the secondary sorting key
type StakerStatsByStakerPagination struct {
	StakerPkHex string `json:"staker_pk_hex"`
	ActiveTvl   int64  `json:"active_tvl"`
}

func DecodeStakerStatsByStakerPaginationToken(token string) (*StakerStatsByStakerPagination, error) {
	tokenBytes, err := base64.URLEncoding.DecodeString(token)
	if err != nil {
		return nil, err
	}
	var d StakerStatsByStakerPagination
	err = json.Unmarshal(tokenBytes, &d)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (d *StakerStatsByStakerPagination) GetPaginationToken() (string, error) {
	tokenBytes, err := json.Marshal(d)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(tokenBytes), nil
}

func BuildStakerStatsByStakerPaginationToken(d StakerStatsDocument) (string, error) {
	page := &StakerStatsByStakerPagination{
		StakerPkHex: d.StakerPkHex,
		ActiveTvl:   d.ActiveTvl,
	}
	token, err := page.GetPaginationToken()
	if err != nil {
		return "", err
	}
	return token, nil
}
