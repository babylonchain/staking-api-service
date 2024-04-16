package model

const StatsLockCollection = "stats_lock"
const OverallStatsCollection = "overall_stats"
const FinalityProviderStatsCollection = "finality_providers_stats"

// StatsLockDocument represents the document in the stats lock collection
// It's used as a lock to prevent concurrent stats calculation for the same staking tx hash
// As well as to prevent the same staking tx hash + txType to be processed multiple times
// The already processed stats will be marked as true in the document
type StatsLockDocument struct {
	Id            string `bson:"_id"`
	OverallStats  bool   `bson:"overall_stats"`
	StakerStats   bool   `bson:"staker_stats"`
	FinalityStats bool   `bson:"finality_stats"`
}

func NewStatsLockDocument(
	id string, overallStats, stakerStats, finalityStats bool,
) *StatsLockDocument {
	return &StatsLockDocument{
		Id:            id,
		OverallStats:  overallStats,
		StakerStats:   stakerStats,
		FinalityStats: finalityStats,
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
