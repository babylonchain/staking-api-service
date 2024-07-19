package types

import "encoding/json"

type UTXORequest struct {
	Txid string `json:"txid"`
	Vout int    `json:"vout"`
}

type SafeUTXO struct {
	TxId        string `json:"txid"`
	Inscription bool   `json:"inscription"`
}

type OrdinalsOutputResponse struct {
	Transaction  string          `json:"transaction"`
	Inscriptions []string        `json:"inscriptions"`
	Runes        json.RawMessage `json:"runes"`
}
