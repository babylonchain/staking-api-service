package services

import (
	"context"

	"github.com/babylonchain/staking-api-service/internal/types"
	"github.com/rs/zerolog/log"
)

type SafeUTXOPublic struct {
	TxId        string `json:"txid"`
	Inscription bool   `json:"inscription"`
}

func (s *Services) VerifyUTXOs(ctx context.Context, utxos []types.UTXORequest) ([]*SafeUTXOPublic, *types.Error) {
	result, err := s.verifyViaOrdinalService(ctx, utxos)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to verify ordinals via ordinals service")
		// TODO: Add metrics

		// TODO: Add fallback to unisat
		return nil, err
	}

	return result, nil
}

func (s *Services) verifyViaOrdinalService(ctx context.Context, utxos []types.UTXORequest) ([]*SafeUTXOPublic, *types.Error) {
	var results []*SafeUTXOPublic

	outputs, err := s.Clients.Ordinals.FetchUTXOInfos(ctx, utxos)
	if err != nil {
		return nil, err
	}

	for _, output := range outputs {
		hasInscription := false

		// Check if Runes is not an empty JSON object
		if len(output.Runes) > 0 && string(output.Runes) != "{}" {
			hasInscription = true
		} else if len(output.Inscriptions) > 0 { // Check if Inscriptions is not empty
			hasInscription = true
		}
		results = append(results, &SafeUTXOPublic{
			TxId:        output.Transaction,
			Inscription: hasInscription,
		})
	}

	return results, nil
}
