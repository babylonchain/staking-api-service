package client

const (
	ActiveStakingQueueName   string = "active_staking_queue"
	UnbondStakingQueueName   string = "unbond_staking_queue"
	WithdrawStakingQueueName string = "withdraw_staking_queue"
)

const (
	ActiveStakingEventType   EventType = 1
	UnbondStakingEventType   EventType = 2
	WithdrawStakingEventType EventType = 3
)

type EventType int

const (
	UnbondingType UnbondType = "UNBONDING"
	ExpireType    UnbondType = "EXPIRE"
)

type UnbondType string

type ActiveStakingEvent struct {
	EventType               EventType `json:"event_type"` // always 1
	StakingTxHex            string    `json:"staking_tx_hex"`
	StakerPkHex             string    `json:"staker_pk_hex"`
	FinalityProviderPkHex   string    `json:"finality_provider_pk_hex"`
	StakingValue            uint64    `json:"staking_value"`
	StakingStartBlockHeight uint64    `json:"staking_start_block_height"`
	StakingTimeLock         uint16    `json:"staking_timelock"`
}

type UnbondStakingEvent struct {
	EventType     EventType  `json:"event_type"` // always 2
	StakingTxHash string     `json:"staking_tx_hash"`
	UnbondType    UnbondType `json:"unbond_type"`
}

type WithdrawStakingEvent struct {
	EventType     EventType `json:"event_type"` // always 3
	StakingTxHash string    `json:"staking_tx_hash"`
}
