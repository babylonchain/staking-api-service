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

type SafeUTXOResponse struct {
	Data  []SafeUTXO    `json:"data"`
	Error []ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	TxId      string `json:"txid,omitempty"`
	Message   string `json:"message"`
	Status    int    `json:"status"`
	ErrorCode string `json:"error_code"`
}

type OrdinalOutputResponse struct {
	Value        int             `json:"value"`
	ScriptPubKey string          `json:"script_pubkey"`
	Address      string          `json:"address"`
	Transaction  string          `json:"transaction"`
	SatRanges    interface{}     `json:"sat_ranges"`
	Inscriptions []string        `json:"inscriptions"`
	Runes        json.RawMessage `json:"runes"`
}
