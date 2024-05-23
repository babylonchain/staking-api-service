package utils

import (
	"encoding/hex"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

func isValidPk(pubKeyStr string) bool {
	// check if the public key string is decodable to bytes
	pubKeyBytes, err := hex.DecodeString(pubKeyStr)
	if err != nil {
		return false
	}

	// check if the public key can be parsed
	_, err = btcec.ParsePubKey(pubKeyBytes)
	return err == nil
}

func isValidAddress(btcAddress string, params *chaincfg.Params) bool {
	// check if address has a valid format
	decodedAddr, err := btcutil.DecodeAddress(btcAddress, params)
	if err != nil {
		return false
	}

	// Cast the address to AddressSegWit
	segwitAddr, ok := decodedAddr.(*btcutil.AddressSegWit)
	if !ok {
		return false
	}

	// Check if address is for the network we are using
	if !segwitAddr.IsForNet(params) {
		return false
	}

    // Check if it's taproot address
    witnessProg := segwitAddr.WitnessProgram()
    if len(witnessProg) != 32 || segwitAddr.WitnessVersion() != 1 {
        return false
    }

    return true
}

func isValidTxHash(txHash string) bool {
	// Check if the hash is valid
	_, err := chainhash.NewHashFromStr(txHash)
	return err == nil
}
