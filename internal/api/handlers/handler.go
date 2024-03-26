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

type paginationResponse struct {
	NextKey string `json:"next_key"`
}

type PublicResponse[T any] struct {
	Data       T                   `json:"data"`
	Pagination *paginationResponse `json:"pagination,omitempty"`
}

type Result struct {
	Data   interface{}
	Status int
}

// NewResult returns a successful result, with default status code 200
func NewResultWithPagination[T any](data T, pageToken string) *Result {
	res := &PublicResponse[T]{Data: data, Pagination: &paginationResponse{NextKey: pageToken}}
	return &Result{Data: res, Status: http.StatusOK}
}

func NewResult[T any](data T) *Result {
	res := &PublicResponse[T]{Data: data}
	return &Result{Data: res, Status: http.StatusOK}
}

func New(
	ctx context.Context, cfg *config.Config, services *services.Services,
) (*Handler, error) {
	return &Handler{
		config:   cfg,
		services: services,
	}, nil
}
