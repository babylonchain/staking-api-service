package api

import (
	"encoding/json"
	"net/http"

	logger "github.com/rs/zerolog"

	"github.com/babylonchain/staking-api-service/internal/api/handlers"
	"github.com/babylonchain/staking-api-service/internal/observability/metrics"
	"github.com/babylonchain/staking-api-service/internal/types"
)

type ErrorResponse struct {
	ErrorCode string `json:"errorCode"`
	Message   string `json:"message"`
}

func newInternalServiceError() *ErrorResponse {
	return &ErrorResponse{
		ErrorCode: types.InternalServiceError.String(),
		Message:   "Internal service error",
	}
}

func (e *ErrorResponse) Error() string {
	return e.Message
}

func registerHandler(handlerFunc func(*http.Request) (*handlers.Result, *types.Error)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// Set up metrics recording for the endpoint
		timer := metrics.StartHttpRequestDurationTimer(r.URL.Path)

		// Handle the actual business logic
		result, err := handlerFunc(r)

		if err != nil {
			if http.StatusText(err.StatusCode) == "" {
				logger.Ctx(r.Context()).Error().Err(err).Int("status_code", err.StatusCode).Msg("invalid status code")
				err.StatusCode = http.StatusInternalServerError
			}

			errorResponse := &ErrorResponse{
				ErrorCode: string(err.ErrorCode),
				Message:   err.Err.Error(),
			}
			// Log the error
			if err.StatusCode >= http.StatusInternalServerError {
				logger.Ctx(r.Context()).Error().Err(errorResponse).Msg("request failed with 5xx error")
				errorResponse.Message = "Internal service error" // Hide the internal message error from client
			}
			timer(err.StatusCode)
			// terminate the request here
			writeResponse(w, r, err.StatusCode, errorResponse)
			return
		}

		if result == nil || http.StatusText(result.Status) == "" {
			logger.Ctx(r.Context()).Error().Msg("invalid success response, error returned")
			timer(http.StatusInternalServerError)
			// terminate the request here
			writeResponse(w, r, http.StatusInternalServerError, newInternalServiceError())
			return
		}

		defer timer(result.Status)
		writeResponse(w, r, result.Status, result.Data)
	}
}

// Write and return response
func writeResponse(w http.ResponseWriter, r *http.Request, statusCode int, res interface{}) {
	respBytes, err := json.Marshal(res)

	if err != nil {
		logger.Ctx(r.Context()).Err(err).Msg("failed to marshal error response")
		http.Error(w, "Failed to process the request. Please try again later.", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if _, err := w.Write(respBytes); err != nil {
		logger.Ctx(r.Context()).Err(err).Msg("failed to write response")
		metrics.RecordHttpResponseWriteFailure(statusCode)
	}
}
