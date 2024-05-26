package handlers

import (
	"net/http"

	"github.com/babylonchain/staking-api-service/internal/types"
	"github.com/babylonchain/staking-api-service/internal/utils"
)

// GetStakerDelegations @Summary Get staker delegations
// @Description Retrieves delegations for a given staker
// @Produce json
// @Param staker_btc_pk query string true "Staker BTC Public Key"
// @Param pagination_key query string false "Pagination key to fetch the next page of delegations"
// @Success 200 {object} PublicResponse[[]services.DelegationPublic]{array} "List of delegations and pagination token"
// @Failure 400 {object} types.Error "Error: Bad Request"
// @Router /v1/staker/delegations [get]
func (h *Handler) GetStakerDelegations(request *http.Request) (*Result, *types.Error) {
	stakerBtcPk := request.URL.Query().Get("staker_btc_pk")
	if stakerBtcPk == "" {
		return nil, types.NewErrorWithMsg(http.StatusBadRequest, types.BadRequest, "staker_btc_pk is required")
	}

	paginationKey := request.URL.Query().Get("pagination_key")

	delegations, newPaginationKey, err := h.services.DelegationsByStakerPk(request.Context(), stakerBtcPk, paginationKey)
	if err != nil {
		return nil, err
	}

	return NewResultWithPagination(delegations, newPaginationKey), nil
}

// CheckStakerDelegationExist @Summary Check if a staker has an active delegation
// @Description Check if a staker has an active delegation by the staker BTC address (Taproot only)
// @Description Optionally, you can provide a timeframe to check if the delegation is active within the provided timeframe
// @Description The available timeframe is "today" which checks after UTC 12AM of the current day
// @Produce json
// @Param address query string true "Staker BTC address in Taproot format"
// @Param timeframe query string false "Check if the delegation is active within the provided timeframe" Enums(today)
// @Success 200 {object} Result "Result"
// @Failure 400 {object} types.Error "Error: Bad Request"
// @Router /v1/staker/delegation/check [get]
func (h *Handler) CheckStakerDelegationExist(request *http.Request) (*Result, *types.Error) {
	address := request.URL.Query().Get("address")
	if address == "" {
		return nil, types.NewErrorWithMsg(
			http.StatusBadRequest, types.BadRequest, "address is required",
		)
	}
	isValid := utils.IsValidBtcAddress(address, h.config.Server.BTCNetParam)
	if !isValid {
		return nil, types.NewErrorWithMsg(
			http.StatusBadRequest, types.BadRequest, "invalid address",
		)
	}

	afterTimestamp, err := parseTimeframeToAfterTimestamp(request.URL.Query().Get("timeframe"))
	if err != nil {
		return nil, err
	}

	exist, err := h.services.CheckStakerHasActiveDelegationByAddress(
		request.Context(), address, afterTimestamp,
	)
	if err != nil {
		return nil, err
	}

	return NewResult(exist), nil
}

func parseTimeframeToAfterTimestamp(timeframe string) (int64, *types.Error) {
	switch timeframe {
	case "": // We ignore and return 0 if no timeframe is provided
		return 0, nil
	case "today":
		return utils.GetTodayStartTimestampInSeconds(), nil
	default:
		return 0, types.NewErrorWithMsg(
			http.StatusBadRequest, types.BadRequest, "invalid timeframe value",
		)
	}
}
