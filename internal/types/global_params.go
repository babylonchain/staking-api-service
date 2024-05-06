package types

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"

	"github.com/babylonchain/babylon/btcstaking"
	"github.com/btcsuite/btcd/btcec/v2"
)

type VersionedGlobalParams struct {
	Version          uint64   `json:"version"`
	ActivationHeight uint64   `json:"activation_height"`
	StakingCap       uint64   `json:"staking_cap"`
	Tag              string   `json:"tag"`
	CovenantPks      []string `json:"covenant_pks"`
	CovenantQuorum   uint64   `json:"covenant_quorum"`
	UnbondingTime    uint64   `json:"unbonding_time"`
	UnbondingFee     uint64   `json:"unbonding_fee"`
	MaxStakingAmount uint64   `json:"max_staking_amount"`
	MinStakingAmount uint64   `json:"min_staking_amount"`
	MaxStakingTime   uint64   `json:"max_staking_time"`
	MinStakingTime   uint64   `json:"min_staking_time"`
}

type GlobalParams struct {
	Versions []*VersionedGlobalParams `json:"versions"`
}

func NewGlobalParams(filePath string) (*GlobalParams, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var globalParams GlobalParams
	err = json.Unmarshal(data, &globalParams)
	if err != nil {
		return nil, err
	}
	err = Validate(&globalParams)
	if err != nil {
		return nil, err
	}

	return &globalParams, nil
}

// parseCovenantPubKeyFromHex parses public key string to btc public key
// the input should be 33 bytes
func parseCovenantPubKeyFromHex(pkStr string) (*btcec.PublicKey, error) {
	pkBytes, err := hex.DecodeString(pkStr)
	if err != nil {
		return nil, err
	}

	pk, err := btcec.ParsePubKey(pkBytes)
	if err != nil {
		return nil, err
	}

	return pk, nil
}

// Validate the global params
func Validate(g *GlobalParams) error {
	if len(g.Versions) == 0 {
		return fmt.Errorf("global params must have at least one version")
	}

	// Loop through the versions and validate each one
	var previousParams *VersionedGlobalParams
	for _, p := range g.Versions {
		tagDecoded, err := hex.DecodeString(p.Tag)

		if err != nil {
			return fmt.Errorf("invalid tag: %w", err)
		}

		if len(tagDecoded) != btcstaking.MagicBytesLen {
			return fmt.Errorf("invalid tag length, expected %d, got %d", btcstaking.MagicBytesLen, len(tagDecoded))
		}

		if len(p.CovenantPks) == 0 {
			return fmt.Errorf("empty covenant public keys")
		}
		if p.CovenantQuorum > uint64(len(p.CovenantPks)) {
			return fmt.Errorf("covenant quorum cannot be more than the amount of covenants")
		}

		for _, covPk := range p.CovenantPks {
			_, err := parseCovenantPubKeyFromHex(covPk)
			if err != nil {
				return fmt.Errorf("invalid covenant public key %s: %w", covPk, err)
			}
		}
		if p.MaxStakingAmount <= p.MinStakingAmount {
			return fmt.Errorf("max-staking-amount must be larger than min-staking-amount")
		}

		if p.MaxStakingTime <= p.MinStakingTime {
			return fmt.Errorf("max-staking-time must be larger than min-staking-time")
		}

		if p.ActivationHeight <= 0 {
			return fmt.Errorf("activation height should be positive")
		}
		if p.StakingCap <= 0 {
			return fmt.Errorf("staking cap should be positive")
		}
		// Check previous parameters conditions
		if previousParams != nil {
			if p.Version != previousParams.Version+1 {
				return fmt.Errorf("versions should be monotonically increasing by 1")
			}
			if p.StakingCap < previousParams.StakingCap {
				return fmt.Errorf("staking cap cannot be decreased in later versions")
			}
			if p.ActivationHeight < previousParams.ActivationHeight {
				return fmt.Errorf("activation height cannot be overlapping between earlier and later versions")
			}
		}
		previousParams = p
	}
	return nil
}
