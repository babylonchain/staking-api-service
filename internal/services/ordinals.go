package services

import (
	"context"
	"fmt"

	"github.com/babylonchain/staking-api-service/internal/clients/unisat"
	"github.com/babylonchain/staking-api-service/internal/types"
	"github.com/rs/zerolog/log"
)

type SafeUTXOPublic struct {
	TxId        string `json:"txid"`
	Inscription bool   `json:"inscription"`
}

func (s *Services) VerifyUTXOs(
	ctx context.Context, utxos []types.UTXORequest, address string,
) ([]*SafeUTXOPublic, *types.Error) {
	result, err := s.verifyViaOrdinalService(ctx, utxos)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to verify ordinals via ordinals service")
		// If Unisat service is enabled, try to verify via Unisat service
		if s.cfg.Unisat != nil {
			unisatResult, err := s.verifyViaUnisatService(ctx, address, utxos)
			if err != nil {
				log.Ctx(ctx).Error().Err(err).Msg("failed to verify ordinals via unisat service")
				return nil, err
			}
			return unisatResult, nil
		}
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

func (s *Services) verifyViaUnisatService(ctx context.Context, address string, utxos []types.UTXORequest) ([]*SafeUTXOPublic, *types.Error) {
	cursor := 0
	var inscriptionsUtxos []*unisat.UnisatUtxos

	for {
		inscriptions, err := s.Clients.Unisat.FetchInscriptionsUtxosByAddress(ctx, address, cursor)
		if err != nil {
			return nil, err
		}
		// Append the fetched utxos to the list
		inscriptionsUtxos = append(inscriptionsUtxos, inscriptions...)
		// Stop fetching if the total number of utxos is less than the limit
		if len(inscriptions) < s.cfg.Unisat.Limit {
			break
		}
		// update the cursor for the next fetch
		cursor += s.cfg.Unisat.Limit
	}

	// turn inscriptionsUtxos into a map for easier lookup
	inscriptionsUtxosMap := make(map[string][]*unisat.UnisatInscriptions)
	for _, inscriptionsUtxo := range inscriptionsUtxos {
		key := fmt.Sprintf("%s:%d", inscriptionsUtxo.TxId, inscriptionsUtxo.Vout)
		inscriptionsUtxosMap[key] = inscriptionsUtxo.Inscriptions
	}

	var results []*SafeUTXOPublic
	for _, utxo := range utxos {
		key := fmt.Sprintf("%s:%d", utxo.Txid, utxo.Vout)
		_, ok := inscriptionsUtxosMap[key]
		results = append(results, &SafeUTXOPublic{
			TxId:        utxo.Txid,
			Inscription: ok,
		})
	}
	return results, nil
}
