package utils

import (
	"bytes"
	"encoding/hex"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"
	"github.com/btcsuite/btcd/wire"
)

func GetBtcTxFromHex(txHex string) (*wire.MsgTx, error) {
	txBytes, err := hex.DecodeString(txHex)
	if err != nil {
		return nil, err
	}

	r := bytes.NewReader(txBytes)

	var tx wire.MsgTx
	if err := tx.Deserialize(r); err != nil {
		return nil, err
	}

	return &tx, nil
}

func GetBtcPkFromHex(pkHex string) (*btcec.PublicKey, error) {
	pkBytes, err := hex.DecodeString(pkHex)
	if err != nil {
		return nil, err
	}

	return schnorr.ParsePubKey(pkBytes)
}

func BtcPkToHex(pk *btcec.PublicKey) string {
	return hex.EncodeToString(schnorr.SerializePubKey(pk))
}

func BtcPksToStrings(pks []*btcec.PublicKey) []string {
	btcPkStrings := make([]string, len(pks))
	for i, pk := range pks {
		btcPkStrings[i] = BtcPkToHex(pk)
	}

	return btcPkStrings
}

func GetBtcPksFromStrings(pkStrings []string) ([]*btcec.PublicKey, error) {
	pks := make([]*btcec.PublicKey, len(pkStrings))
	for i, pkStr := range pkStrings {
		pk, err := GetBtcPkFromHex(pkStr)
		if err != nil {
			return nil, err
		}
		pks[i] = pk
	}

	return pks, nil
}
