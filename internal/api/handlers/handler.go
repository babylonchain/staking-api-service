package handlers

import (
	"context"
	"net/http"

	"github.com/babylonchain/staking-api-service/internal/config"
)

type Handler struct {
	config *config.Config
	// Other dependencies to be added here
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
	ctx context.Context, cfg *config.Config,
) (*Handler, error) {
	// Set up the middlewares

	return &Handler{
		config: cfg,
	}, nil
}
