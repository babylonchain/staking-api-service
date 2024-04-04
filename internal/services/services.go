package services

import (
	"context"
	"net/http"

	"github.com/rs/zerolog/log"

	"github.com/babylonchain/staking-api-service/internal/config"
	"github.com/babylonchain/staking-api-service/internal/db"
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
	dbClient, err := db.New(ctx, cfg.Db)
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

// ProcessStakingStatsCalculation calculates the staking stats and updates the database.
// This method tolerate duplicated calls, only the first call will be processed.
func (s *Services) ProcessStakingStatsCalculation(ctx context.Context, eventMessage queue.EventMessage) error {
	return nil
}

func (s *Services) SaveUnprocessableMessages(ctx context.Context, messageBody, receipt string) error {
	err := s.DbClient.SaveUnprocessableMessage(ctx, messageBody, receipt)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("error while saving unprocessable message")
		return types.NewErrorWithMsg(http.StatusInternalServerError, types.InternalServiceError, "error while saving unprocessable message")
	}
	return nil
}
