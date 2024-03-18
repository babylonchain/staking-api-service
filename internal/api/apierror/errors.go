package apierror

import (
	"errors"
	"net/http"
)

type ErrorCode string

// ApiError represents an error with an HTTP status code and an application-specific error code.
type ApiError struct {
	Err        error
	StatusCode int
	ErrorCode  ErrorCode
}

const UninitializedStatusCode = 0

// We define all API errors here. Note, those error types will be returned in the response for FE
const (
	// 5XX
	InternalServiceError ErrorCode = "INTERNAL_SERVICE_ERROR"
	ValidationError      ErrorCode = "VALIDATION_ERROR"
	NotFound             ErrorCode = "NOT_FOUND"
	BadRequest           ErrorCode = "BAD_REQUEST"
)

func (e *ApiError) Error() string {
	return e.Err.Error()
}

// NewError creates a new ApiError with the provided status code, error code, and underlying error.
// If the status code is not provided (0), it defaults to http.StatusInternalServerError(500).
// If the error code is empty, it defaults to INTERNAL_SERVICE_ERROR.
func NewError(statusCode int, errorCode ErrorCode, err error) *ApiError {
	if statusCode == UninitializedStatusCode {
		statusCode = http.StatusInternalServerError
	}
	if errorCode == "" {
		errorCode = InternalServiceError
	}
	return &ApiError{
		StatusCode: statusCode,
		ErrorCode:  errorCode,
		Err:        err,
	}
}

func NewErrorWithMsg(statusCode int, errorCode ErrorCode, msg string) *ApiError {
	return NewError(statusCode, errorCode, errors.New(msg))
}

func NewInternalServiceError(err error) *ApiError {
	return &ApiError{
		StatusCode: http.StatusInternalServerError,
		ErrorCode:  InternalServiceError,
		Err:        err,
	}
}
