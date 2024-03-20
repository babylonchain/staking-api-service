package services

import (
	"context"

	"github.com/rs/zerolog/log"

	"github.com/babylonchain/staking-api-service/internal/config"
	"github.com/babylonchain/staking-api-service/internal/db"
)

// Service layer contains the business logic and is used to interact with
// the database and other external clients (if any).
type Services struct {
	DbClient db.DBClient
	cfg      *config.Config
}

func New(ctx context.Context, cfg *config.Config) (*Services, error) {
	dbClient, err := db.New(ctx, cfg.Db.DbName, cfg.Db.Address)
	if err != nil {
		log.Ctx(ctx).Fatal().Err(err).Msg("error while creating db client")
		return nil, err
	}
	return &Services{
		DbClient: dbClient,
		cfg:      cfg,
	}, nil
}

// DoHealthCheck checks the health of the services by ping the database.
func (s *Services) DoHealthCheck(ctx context.Context) error {
	return s.DbClient.Ping(ctx)
}
