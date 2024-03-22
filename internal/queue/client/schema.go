package client

const (
	ActiveStakingQueueName    string = "active_staking_queue"
	UnbondingStakingQueueName string = "unbonding_staking_queue"
	WithdrawStakingQueueName  string = "withdraw_staking_queue"
	ExpiredStakingQueueName   string = "expired_staking_queue"
)

const (
	ActiveStakingEventType    EventType = 1
	UnbondingStakingEventType EventType = 2
	WithdrawStakingEventType  EventType = 3
	ExpiredStakingEventType   EventType = 4
)

type EventType int

type ActiveStakingEvent struct {
	EventType             EventType `json:"event_type"` // always 1. ActiveStakingEventType
	StakingTxHex          string    `json:"staking_tx_hex"`
	StakerPkHex           string    `json:"staker_pk_hex"`
	FinalityProviderPkHex string    `json:"finality_provider_pk_hex"`
	StakingValue          uint64    `json:"staking_value"`
	StakingStartkHeight   uint64    `json:"staking_start_height"`
	StakingTimeLock       uint16    `json:"staking_timelock"`
}

type UnbondingStakingEvent struct {
	EventType            EventType `json:"event_type"` // always 2. UnbondingStakingEventType
	StakingTxHash        string    `json:"staking_tx_hash"`
	UnbondingStartHeight uint64    `json:"unbonding_start_height"`
	UnbondingTimeLock    uint16    `json:"unbonding_timelock"`
}

type WithdrawStakingEvent struct {
	EventType     EventType `json:"event_type"` // always 3. WithdrawStakingEventType
	StakingTxHash string    `json:"staking_tx_hash"`
}

type ExpiredStakingEvent struct {
	EventType     EventType `json:"event_type"` // always 4. ExpiredStakingEventType
	StakingTxHash string    `json:"staking_tx_hash"`
}
