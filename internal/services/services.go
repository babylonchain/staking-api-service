package services

import (
	"context"
	"net/http"

	"github.com/rs/zerolog/log"

	"github.com/babylonchain/staking-api-service/internal/config"
	"github.com/babylonchain/staking-api-service/internal/db"
	"github.com/babylonchain/staking-api-service/internal/db/model"
	"github.com/babylonchain/staking-api-service/internal/types"

	queue "github.com/babylonchain/staking-queue-client/client"
)

// Service layer contains the business logic and is used to interact with
// the database and other external clients (if any).
type Services struct {
	DbClient db.DBClient
	cfg      *config.Config
	params   *types.GlobalParams
}

func New(ctx context.Context, cfg *config.Config, globalParams *types.GlobalParams) (*Services, error) {
	dbClient, err := db.New(ctx, cfg.Db.DbName, cfg.Db.Address)
	if err != nil {
		log.Ctx(ctx).Fatal().Err(err).Msg("error while creating db client")
		return nil, err
	}
	return &Services{
		DbClient: dbClient,
		cfg:      cfg,
		params:   globalParams,
	}, nil
}

// DoHealthCheck checks the health of the services by ping the database.
func (s *Services) DoHealthCheck(ctx context.Context) error {
	return s.DbClient.Ping(ctx)
}

// SaveActiveStakingDelegation saves the active staking delegation to the database.
func (s *Services) SaveActiveStakingDelegation(
	ctx context.Context, txHashHex string, stakerPkHex string,
	finalityProviderPkHex string, value uint64, startHeight uint64, timeLock uint64,
) error {
	err := s.DbClient.SaveActiveStakingDelegation(
		ctx,
		txHashHex, stakerPkHex, finalityProviderPkHex, value, startHeight, timeLock,
	)
	if err != nil {
		if ok := db.IsDuplicateKeyError(err); ok {
			log.Warn().Err(err).Msg("Skip the active staking event as it already exists in the database")
			// TODO: Add metrics for duplicate active staking events
			return nil
		}
		log.Error().Err(err).Msg("Failed to save active staking delegation")
		return types.NewInternalServiceError(err)
	}
	return nil
}

// ProcessStakingStatsCalculation calculates the staking stats and updates the database.
// This method tolerate duplicated calls, only the first call will be processed.
func (s *Services) ProcessStakingStatsCalculation(ctx context.Context, eventMessage queue.EventMessage) error {
	return nil
}

func (s *Services) DelegationsByStakerPk(ctx context.Context, stakerPk string, pageToken string) ([]DelegationPublic, string, *types.Error) {
	resultMap, err := s.DbClient.FindDelegationsByStakerPk(ctx, stakerPk, pageToken)
	if err != nil {
		if db.IsInvalidPaginationTokenError(err) {
			log.Warn().Err(err).Msg("Invalid pagination token when fetching delegations by staker pk")
			return nil, "", types.NewError(http.StatusBadRequest, types.BadRequest, err)
		}
		log.Error().Err(err).Msg("Failed to find delegations by staker pk")
		return nil, "", types.NewInternalServiceError(err)
	}
	var delegations []DelegationPublic
	for _, d := range resultMap.Data {
		delegations = append(delegations, fromDelegationDocument(d))
	}
	return delegations, resultMap.PaginationToken, nil
}

type DelegationPublic struct {
	StakingTxHashHex      string `json:"staking_tx_hash_hex"`
	StakerPkHex           string `json:"staker_pk_hex"`
	FinalityProviderPkHex string `json:"finality_provider_pk_hex"`
	StakingValue          uint64 `json:"staking_value"`
	TimeLockExpireHeight  uint64 `json:"time_lock_expire"`
	State                 string `json:"state"`
}

func fromDelegationDocument(d model.DelegationDocument) DelegationPublic {
	return DelegationPublic{
		StakingTxHashHex:      d.StakingTxHashHex,
		FinalityProviderPkHex: d.FinalityProviderPkHex,
		StakerPkHex:           d.StakerPkHex,
		StakingValue:          d.StakingValue,
		TimeLockExpireHeight:  d.StakingStartHeight + d.StakingTimeLock,
		State:                 d.State.ToString(),
	}
}
