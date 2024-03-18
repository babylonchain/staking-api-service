package api

import (
	"encoding/json"
	"errors"
	"net/http"

	logger "github.com/rs/zerolog"

	"github.com/babylonchain/staking-api-service/internal/api/apierror"
	"github.com/babylonchain/staking-api-service/internal/api/handlers"
	"github.com/babylonchain/staking-api-service/internal/observability/metrics"
)

type ErrorResponse struct {
	ErrorCode string `json:"errorCode"`
	Message   string `json:"message"`
}

func (e *ErrorResponse) Error() string {
	return e.Message
}

func registerHandler(handlerFunc func(*http.Request) (*handlers.Result, error)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// Set up metrics recording for the endpoint
		timer := metrics.StartHttpRequestDurationTimer(r.URL.Path)

		// Handle the actual business logic
		res, err := handlerFunc(r)
		var statusCode int
		var result interface{}

		if err != nil {
			// Catch the our own error type. Otherwise, all errors are internal service error
			var apiErr *apierror.ApiError
			var errorResponse *ErrorResponse

			if errors.As(err, &apiErr) {
				apiErr = err.(*apierror.ApiError)
				errorResponse = &ErrorResponse{
					ErrorCode: string(apiErr.ErrorCode),
					Message:   apiErr.Err.Error(),
				}
				statusCode = apiErr.StatusCode
			} else {
				logger.Ctx(r.Context()).Err(err).Msg("failed to handle request")
				errorResponse = &ErrorResponse{
					ErrorCode: string(apierror.InternalServiceError),
					Message:   err.Error(),
				}
			}
			// Validate the status code
			if http.StatusText(statusCode) == "" {
				logger.Ctx(r.Context()).Error().Int("status_code", statusCode).Msg("invalid status code")
				statusCode = http.StatusInternalServerError
			}
			// Log the error
			if statusCode >= http.StatusInternalServerError {
				logger.Ctx(r.Context()).Error().Err(errorResponse).Msg("request failed with 5xx error")
				errorResponse.Message = "Internal server error" // Hide the internal message error from client
			}
			result = errorResponse
		} else {
			result = res.Data
			statusCode = res.Status
		}

		defer timer(statusCode)

		respBytes := []byte{}
		if result != nil {
			respBytes, err = json.Marshal(result)

			if err != nil {
				logger.Ctx(r.Context()).Err(err).Msg("failed to marshal response")
				http.Error(w, "Failed to process the request. Please try again later.", http.StatusInternalServerError)
				return
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		w.Write(respBytes) // nolint:errcheck

	}
}
