package utils

import (
	"encoding/hex"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

func IsValidPk(pubKeyStr string) bool {
	// check if the public key string is decodable to bytes
	pubKeyBytes, err := hex.DecodeString(pubKeyStr)
	if err != nil {
		return false
	}

	// check if the public key can be parsed
	_, err = btcec.ParsePubKey(pubKeyBytes)
	return err == nil
}

// IsValidBtcAddress checks if the provided address is a valid BTC address
// We only support Taproot addresses and native SegWit addresses
func IsValidBtcAddress(btcAddress string, params *chaincfg.Params) bool {
	// Check if address has a valid format
	decodedAddr, err := btcutil.DecodeAddress(btcAddress, params)
	if err != nil {
		return false
	}

	// Check if address is for the network we are using
	if !decodedAddr.IsForNet(params) {
		return false
	}
	// Check if it's either a native SegWit (P2WPKH) or Taproot address
	switch decodedAddr.(type) {
	case *btcutil.AddressWitnessPubKeyHash:
		// Native SegWit (P2WPKH)
		return true
	case *btcutil.AddressTaproot:
		// Taproot address
		return true
	default:
		return false
	}
}

func IsValidTxHash(txHash string) bool {
	// Check if the hash is valid
	_, err := chainhash.NewHashFromStr(txHash)
	return err == nil
}
