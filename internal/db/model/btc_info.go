package model

const LatestBtcInfoId = "latest"

type BtcInfo struct {
	ID                   string `bson:"_id"`
	BtcHeight            uint64 `bson:"btc_height"`
	UnconfirmedActiveTvl uint64 `bson:"unconfirmed_active_tvl"`
}
