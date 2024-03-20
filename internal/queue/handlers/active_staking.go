package handlers

import (
	"context"

	"github.com/rs/zerolog/log"
)

func (h *QueueHandler) ActiveStakingHandler(ctx context.Context, messageBody string) error {
	log.Info().Msgf("Received message from active staking queue: %s", messageBody)

	// TODO: implement the business logic for processing the message from the active staking queue
	h.Services.DoHealthCheck(ctx)
	return nil
}
