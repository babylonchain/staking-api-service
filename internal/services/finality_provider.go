package services

import (
	"context"
	"net/http"
	"sort"

	"github.com/babylonchain/staking-api-service/internal/types"
	"github.com/rs/zerolog/log"
)

type FpDescriptionPublic struct {
	Moniker         string `json:"moniker"`
	Identity        string `json:"identity"`
	Website         string `json:"website"`
	SecurityContact string `json:"security_contact"`
	Details         string `json:"details"`
}

type FpDetailsPublic struct {
	Description       FpDescriptionPublic `json:"description"`
	Commission        string              `json:"commission"`
	BtcPk             string              `json:"btc_pk"`
	ActiveTvl         uint64              `json:"active_tvl"`
	TotalTvl          uint64              `json:"total_tvl"`
	ActiveDelegations uint64              `json:"active_delegations"`
	TotalDelegations  uint64              `json:"total_delegations"`
}

func (s *Services) GetActiveFinalityProviders(ctx context.Context) ([]FpDetailsPublic, *types.Error) {
	fpParams := s.GetFinalityProvidersFromGlobalParams()
	if len(fpParams) == 0 {
		log.Ctx(ctx).Error().Msg("No finality providers found from global params")
		return nil, types.NewErrorWithMsg(http.StatusInternalServerError, types.InternalServiceError, "No finality providers found from global params")
	}

	var fpBtcPks []string
	for _, fp := range fpParams {
		fpBtcPks = append(fpBtcPks, fp.BtcPk)
	}

	finalityProvidersMap, err := s.DbClient.FindFinalityProvidersByPkHex(ctx, fpBtcPks)
	if err != nil {
		// We don't want to return an error here in case of DB error.
		// we will continue the process with the data we have from global params as a fallback.
		log.Ctx(ctx).Error().Err(err).Msg("Error while fetching finality providers from DB")
		// TODO: Metric for this error and alerting
	}

	var finalityProviderDetailsPublic []FpDetailsPublic

	for _, fp := range fpParams {
		// Default values being set for ActiveTvl, TotalTvl, ActiveDelegations, TotalDelegations
		// This could happen if our system has never processed any staking tx events associated to this finality provider
		detail := FpDetailsPublic{
			Description:       fp.Description,
			Commission:        fp.Commission,
			BtcPk:             fp.BtcPk,
			ActiveTvl:         0,
			TotalTvl:          0,
			ActiveDelegations: 0,
			TotalDelegations:  0,
		}

		if finalityProvider, ok := finalityProvidersMap[fp.BtcPk]; ok {
			detail.ActiveTvl = finalityProvider.ActiveTvl
			detail.TotalTvl = finalityProvider.TotalTvl
			detail.ActiveDelegations = finalityProvider.ActiveDelegations
			detail.TotalDelegations = finalityProvider.TotalDelegations
		} else if !ok {
			log.Ctx(ctx).Warn().Str("btc_pk", fp.BtcPk).Msg("Finality provider not found in DB")
		}
		finalityProviderDetailsPublic = append(finalityProviderDetailsPublic, detail)
	}

	// Sort the finalityProviderDetailsPublic slice by ActiveTvl in descending order
	sort.SliceStable(finalityProviderDetailsPublic, func(i, j int) bool {
		return finalityProviderDetailsPublic[i].ActiveTvl > finalityProviderDetailsPublic[j].ActiveTvl
	})

	return finalityProviderDetailsPublic, nil
}
