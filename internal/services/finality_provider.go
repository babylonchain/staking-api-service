package services

import (
	"context"
	"net/http"

	"github.com/babylonchain/staking-api-service/internal/db"
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

func EmptyFpDescriptionPublic() FpDescriptionPublic {
	return FpDescriptionPublic{
		Moniker:         "",
		Identity:        "",
		Website:         "",
		SecurityContact: "",
		Details:         "",
	}
}

type FpDetailsPublic struct {
	Description       *FpDescriptionPublic `json:"description"`
	Commission        string               `json:"commission"`
	BtcPk             string               `json:"btc_pk"`
	ActiveTvl         int64                `json:"active_tvl"`
	TotalTvl          int64                `json:"total_tvl"`
	ActiveDelegations int64                `json:"active_delegations"`
	TotalDelegations  int64                `json:"total_delegations"`
}

func (s *Services) GetFinalityProviders(ctx context.Context, page string) ([]FpDetailsPublic, string, *types.Error) {
	fpParams := s.GetFinalityProvidersFromGlobalParams()
	if len(fpParams) == 0 {
		log.Ctx(ctx).Error().Msg("No finality providers found from global params")
		return nil, "", types.NewErrorWithMsg(http.StatusInternalServerError, types.InternalServiceError, "No finality providers found from global params")
	}
	// Convert the fpParams slice to a map with the BtcPk as the key
	fpParamsMap := make(map[string]*FpParamsPublic)
	for _, fp := range fpParams {
		fpParamsMap[fp.BtcPk] = &fp
	}

	resultMap, err := s.DbClient.FindFinalityProviderStats(ctx, page)
	if err != nil {
		if db.IsInvalidPaginationTokenError(err) {
			log.Ctx(ctx).Warn().Err(err).Msg("Invalid pagination token when fetching finality providers")
			return nil, "", types.NewError(http.StatusBadRequest, types.BadRequest, err)
		}
		// We don't want to return an error here in case of DB error.
		// we will continue the process with the data we have from global params as a fallback.
		// TODO: Add metric for this error and alerting
		log.Ctx(ctx).Error().Err(err).Msg("Error while fetching finality providers from DB")
		// Return the finality providers from global params as a fallback
		return buildFallbackFpDetailsPublic(fpParams), "", nil
	}
	// If no finality providers are found in the DB,
	// return the finality providers from global params as a fallback
	if len(resultMap.Data) == 0 {
		return buildFallbackFpDetailsPublic(fpParams), "", nil
	}

	var finalityProviderDetailsPublic []FpDetailsPublic
	for _, fp := range resultMap.Data {
		var paramsPublic *FpParamsPublic
		if fpParamsMap[fp.FinalityProviderPkHex] != nil {
			paramsPublic = fpParamsMap[fp.FinalityProviderPkHex]
		} else {
			paramsPublic = &FpParamsPublic{
				Description: EmptyFpDescriptionPublic(),
				Commission:  "",
				BtcPk:       fp.FinalityProviderPkHex,
			}
		}
		detail := FpDetailsPublic{
			Description:       &paramsPublic.Description,
			Commission:        paramsPublic.Commission,
			BtcPk:             fp.FinalityProviderPkHex,
			ActiveTvl:         fp.ActiveTvl,
			TotalTvl:          fp.TotalTvl,
			ActiveDelegations: fp.ActiveDelegations,
			TotalDelegations:  fp.TotalDelegations,
		}
		finalityProviderDetailsPublic = append(finalityProviderDetailsPublic, detail)
	}

	return finalityProviderDetailsPublic, resultMap.PaginationToken, nil
}

func buildFallbackFpDetailsPublic(fpParams []FpParamsPublic) []FpDetailsPublic {
	var finalityProviderDetailsPublic []FpDetailsPublic
	for _, fp := range fpParams {
		detail := FpDetailsPublic{
			Description:       &fp.Description,
			Commission:        fp.Commission,
			BtcPk:             fp.BtcPk,
			ActiveTvl:         0,
			TotalTvl:          0,
			ActiveDelegations: 0,
			TotalDelegations:  0,
		}
		finalityProviderDetailsPublic = append(finalityProviderDetailsPublic, detail)
	}
	return finalityProviderDetailsPublic
}
