package types

type ExpiredTxType string

const (
	ActiveType    ExpiredTxType = "active"
	UnbondingType ExpiredTxType = "unbonding"
)

func (s ExpiredTxType) ToString() string {
	return string(s)
}

func ExpiredTxTypeFromString(s string) ExpiredTxType {
	switch s {
	case ActiveType.ToString():
		return ActiveType
	case UnbondingType.ToString():
		return UnbondingType
	default:
		return ""
	}
}
