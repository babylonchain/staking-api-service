package tests

import (
	"encoding/hex"
	"fmt"
	"math/rand"
	"testing"

	bbndatagen "github.com/babylonchain/babylon/testutil/datagen"
	"github.com/babylonchain/staking-api-service/internal/types"
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"
)

const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func generateRandomFinalityProviderDetail(t *testing.T, r *rand.Rand, numOfFps uint64) []types.FinalityProviderDetails {
	var finalityProviders []types.FinalityProviderDetails

	for i := uint64(0); i < numOfFps; i++ {
		fpPkInHex, err := randomPk()
		if err != nil {
			t.Fatalf("failed to generate random public key: %v", err)
		}

		randomStr := randomString(r, 10)
		finalityProviders = append(finalityProviders, types.FinalityProviderDetails{
			Description: types.FinalityProviderDescription{
				Moniker:         "Moniker" + randomStr,
				Identity:        "Identity" + randomStr,
				Website:         "Website" + randomStr,
				SecurityContact: "SecurityContact" + randomStr,
				Details:         "Details" + randomStr,
			},
			Commission: fmt.Sprintf("%f", randomFloat64(r)),
			BtcPk:      fpPkInHex,
		})
	}
	return finalityProviders
}

// randomFloat64 generates a random float64 value greater than 0.
func randomFloat64(r *rand.Rand) float64 {
	for {
		f := r.Float64() // Generate a random float64
		if f > 0 {
			return f
		}
		// If f is 0 (extremely rare), regenerate
	}
}

func randomPk() (string, error) {
	fpPirvKey, err := btcec.NewPrivateKey()
	if err != nil {
		return "", err
	}
	fpPk := fpPirvKey.PubKey()
	return hex.EncodeToString(schnorr.SerializePubKey(fpPk)), nil
}

// randomString generates a random alphanumeric string of length n.
func randomString(r *rand.Rand, n int) string {
	result := make([]byte, n)
	letterLen := len(letters)
	for i := range result {
		num := r.Int() % letterLen
		result[i] = letters[num]
	}
	return string(result)
}

// randomAmount generates a random BTC amount from 0.1 to 10000
// the returned value is in satoshis
func randomAmount(r *rand.Rand) int64 {
	// Generate a random value range from 0.1 to 10000 BTC
	randomBTC := r.Float64()*(9999.9-0.1) + 0.1
	// convert to satoshi
	return int64(randomBTC*1e8) + 1
}

func attachRandomSeedsToFuzzer(f *testing.F, numOfSeeds int) {
	bbndatagen.AddRandomSeedsToFuzzer(f, uint(numOfSeeds))
}
