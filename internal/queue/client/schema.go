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
	UnbondStakingEventType    EventType = 4
)

type EventType int

type EventMessage interface {
	GetEventType() EventType
	GetStakingTxHash() string
}

type ActiveStakingEvent struct {
	EventType             EventType `json:"event_type"` // always 1. ActiveStakingEventType
	StakingTxHex          string    `json:"staking_tx_hex"`
	StakerPkHex           string    `json:"staker_pk_hex"`
	FinalityProviderPkHex string    `json:"finality_provider_pk_hex"`
	StakingValue          uint64    `json:"staking_value"`
	StakingStartkHeight   uint64    `json:"staking_start_height"`
	StakingTimeLock       uint64    `json:"staking_timelock"`
}

func (e ActiveStakingEvent) GetEventType() EventType {
	return ActiveStakingEventType
}

func (e ActiveStakingEvent) GetStakingTxHash() string {
	return e.StakingTxHex
}

func NewActiveStakingEvent(
	stakingTxHex string,
	stakerPkHex string,
	finalityProviderPkHex string,
	stakingValue uint64,
	stakingStartHeight uint64,
	stakingTimeLock uint64,
) ActiveStakingEvent {
	return ActiveStakingEvent{
		EventType:             ActiveStakingEventType,
		StakingTxHex:          stakingTxHex,
		StakerPkHex:           stakerPkHex,
		FinalityProviderPkHex: finalityProviderPkHex,
		StakingValue:          stakingValue,
		StakingStartkHeight:   stakingStartHeight,
		StakingTimeLock:       stakingTimeLock,
	}
}

type UnbondingStakingEvent struct {
	EventType            EventType `json:"event_type"` // always 2. UnbondingStakingEventType
	StakingTxHash        string    `json:"staking_tx_hash"`
	UnbondingStartHeight uint64    `json:"unbonding_start_height"`
	UnbondingTimeLock    uint16    `json:"unbonding_timelock"`
}

func (e UnbondingStakingEvent) GetEventType() EventType {
	return UnbondingStakingEventType
}

func (e UnbondingStakingEvent) GetStakingTxHash() string {
	return e.StakingTxHash
}

func NewUnbondingStakingEvent(
	stakingTxHash string,
	unbondingStartHeight uint64,
	unbondingTimeLock uint16,
) UnbondingStakingEvent {
	return UnbondingStakingEvent{
		EventType:            UnbondingStakingEventType,
		StakingTxHash:        stakingTxHash,
		UnbondingStartHeight: unbondingStartHeight,
		UnbondingTimeLock:    unbondingTimeLock,
	}
}

type WithdrawStakingEvent struct {
	EventType     EventType `json:"event_type"` // always 3. WithdrawStakingEventType
	StakingTxHash string    `json:"staking_tx_hash"`
}

func (e WithdrawStakingEvent) GetEventType() EventType {
	return WithdrawStakingEventType
}

func (e WithdrawStakingEvent) GetStakingTxHash() string {
	return e.StakingTxHash
}

func NewWithdrawStakingEvent(stakingTxHash string) WithdrawStakingEvent {
	return WithdrawStakingEvent{
		EventType:     WithdrawStakingEventType,
		StakingTxHash: stakingTxHash,
	}
}

type UnbondStakingEvent struct {
	EventType     EventType `json:"event_type"` // always 4. UnbondStakingEventType
	StakingTxHash string    `json:"staking_tx_hash"`
}

func (e UnbondStakingEvent) GetEventType() EventType {
	return UnbondStakingEventType
}

func (e UnbondStakingEvent) GetStakingTxHash() string {
	return e.StakingTxHash
}

func NewUnbondStakingEvent(stakingTxHash string) UnbondStakingEvent {
	return UnbondStakingEvent{
		EventType:     UnbondStakingEventType,
		StakingTxHash: stakingTxHash,
	}
}
