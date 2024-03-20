package handlers

import "github.com/rs/zerolog/log"

func (h *QueueHandler) ActiveStakingHandler(messageBody string) error {
	log.Info().Msgf("Received message from active staking queue: %s", messageBody)
	return nil
}
