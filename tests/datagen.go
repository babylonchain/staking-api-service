package tests

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math/rand"
	"testing"
	"time"

	bbndatagen "github.com/babylonchain/babylon/testutil/datagen"
	"github.com/babylonchain/staking-api-service/internal/types"
	"github.com/babylonchain/staking-queue-client/client"
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
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

func randomPositiveInt(r *rand.Rand, max int) int {
	if max == 1 {
		return 1
	}
	// Generate a random number from 1 to max (inclusive)
	return r.Intn(max) + 1
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

// generate a random height from 1 to maxHeight
// if maxHeight is 0, then we default the max height to 1000000
func randomBtcHeight(r *rand.Rand, maxHeight uint64) uint64 {
	if maxHeight == 1 {
		return 1
	}
	if maxHeight == 0 {
		maxHeight = 1000000
	}
	return uint64(r.Intn(int(maxHeight))) + 1
}

func generateRandomTx(r *rand.Rand) (*wire.MsgTx, string, error) {
	tx := &wire.MsgTx{
		Version: 1,
		TxIn: []*wire.TxIn{
			{
				PreviousOutPoint: wire.OutPoint{
					Hash:  chainhash.HashH(bbndatagen.GenRandomByteArray(r, 10)),
					Index: r.Uint32(),
				},
				SignatureScript: bbndatagen.GenRandomByteArray(r, 10),
				Sequence:        r.Uint32(),
			},
		},
		TxOut: []*wire.TxOut{
			{
				Value:    r.Int63(),
				PkScript: bbndatagen.GenRandomByteArray(r, 80),
			},
		},
		LockTime: 0,
	}
	var buf bytes.Buffer
	if err := tx.Serialize(&buf); err != nil {
		return nil, "", err
	}
	txHex := hex.EncodeToString(buf.Bytes())

	return tx, txHex, nil
}

type TestActiveEventGeneratorOpts struct {
	NumOfEvents     int
	NumberOfFps     int
	NumberOfStakers int
}

// generateRandomActiveStakingEvents generates a random number of active staking events
// with random values for each field.
// default to max 11 events, 11 finality providers, and 11 stakers
func generateRandomActiveStakingEvents(
	t *testing.T, r *rand.Rand, opts *TestActiveEventGeneratorOpts,
) []*client.ActiveStakingEvent {
	var activeStakingEvents []*client.ActiveStakingEvent
	if opts == nil {
		opts = &TestActiveEventGeneratorOpts{
			NumOfEvents:     11,
			NumberOfFps:     11,
			NumberOfStakers: 11,
		}
	}

	var fpPks []string
	for i := 0; i < opts.NumberOfFps; i++ {
		fpPk, err := randomPk()
		if err != nil {
			t.Fatalf("failed to generate random public key for FP: %v", err)
		}
		fpPks = append(fpPks, fpPk)
	}

	var stakerPks []string
	for i := 0; i < opts.NumberOfStakers; i++ {
		stakerPk, err := randomPk()
		if err != nil {
			t.Fatalf("failed to generate random public key for staker: %v", err)
		}
		stakerPks = append(stakerPks, stakerPk)
	}

	for i := 0; i < opts.NumOfEvents; i++ {
		randomFpPk := fpPks[rand.Intn(len(fpPks))]
		randomStakerPk := stakerPks[rand.Intn(len(stakerPks))]
		tx, hex, err := generateRandomTx(r)
		if err != nil {
			t.Fatalf("failed to generate random tx: %v", err)
		}
		activeStakingEvent := &client.ActiveStakingEvent{
			EventType:             client.ActiveStakingEventType,
			StakingTxHashHex:      tx.TxHash().String(),
			StakerPkHex:           randomStakerPk,
			FinalityProviderPkHex: randomFpPk,
			StakingValue:          uint64(randomAmount(r)),
			StakingStartHeight:    randomBtcHeight(r, 0),
			StakingStartTimestamp: time.Now().Unix(),
			StakingTimeLock:       uint64(rand.Intn(100)),
			StakingOutputIndex:    uint64(rand.Intn(100)),
			StakingTxHex:          hex,
			IsOverflow:            rand.Int()%2 == 0,
		}
		activeStakingEvents = append(activeStakingEvents, activeStakingEvent)
	}
	return activeStakingEvents
}
