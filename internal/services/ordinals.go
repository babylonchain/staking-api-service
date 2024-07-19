package services

import (
	"context"
	"encoding/json"

	"github.com/babylonchain/staking-api-service/internal/types"
)

func (s *Services) VerifyUTXOs(ctx context.Context, utxos []types.UTXORequest) ([]types.SafeUTXO, *types.Error) {
	var results []types.SafeUTXO

	outputs, err := s.Clients.Ordinals.FetchUTXOInfos(ctx, utxos)
	if err != nil {
		return nil, err
	}

	for _, output := range outputs {
		var runes []string

		// Check if Runes is not an empty JSON object
		if len(output.Runes) > 0 && string(output.Runes) != "{}" {
			if err := json.Unmarshal(output.Runes, &runes); err != nil {
				continue
			}
		}

		safe := len(output.Inscriptions) == 0 && len(runes) == 0
		results = append(results, types.SafeUTXO{
			TxId:        output.Transaction,
			Inscription: !safe,
		})
	}

	return results, nil
}
