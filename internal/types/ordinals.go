package types

import "encoding/json"

type UTXORequest struct {
	Txid string `json:"txid"`
	Vout int    `json:"vout"`
}

type OrdinalsOutputResponse struct {
	Transaction  string          `json:"transaction"` // same as Txid
	Inscriptions []string        `json:"inscriptions"`
	Runes        json.RawMessage `json:"runes"`
}
