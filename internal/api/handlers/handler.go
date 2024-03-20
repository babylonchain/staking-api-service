package handlers

import (
	"context"
	"net/http"

	"github.com/babylonchain/staking-api-service/internal/config"
	"github.com/babylonchain/staking-api-service/internal/services"
)

type Handler struct {
	config   *config.Config
	services *services.Services
}

type Result struct {
	Data   interface{}
	Status int
}

// NewResult returns a successful result, with default status code 200
func NewResult(data any) *Result {
	return &Result{Data: data, Status: http.StatusOK}
}

func New(
	ctx context.Context, cfg *config.Config, services *services.Services,
) (*Handler, error) {
	return &Handler{
		config:   cfg,
		services: services,
	}, nil
}
