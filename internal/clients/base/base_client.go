package baseclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/babylonchain/staking-api-service/internal/types"
	"github.com/rs/zerolog/log"
)

type BaseClient interface {
	GetBaseURL() string
	GetDefaultRequestTimeout() int
	GetHttpClient() *http.Client
}

type BaseClientOptions struct {
	Timeout int
	Path    string
	Headers map[string]string
}

// Add metrics and logs
func PostRequest[I any, R any](
	ctx context.Context, client BaseClient, opts *BaseClientOptions, input I,
) (*R, *types.Error) {
	url := fmt.Sprintf("%s%s", client.GetBaseURL(), opts.Path)
	timeout := client.GetDefaultRequestTimeout()
	// If timeout is set, use it instead of the default
	if opts.Timeout != 0 {
		timeout = opts.Timeout
	}
	// Set a timeout for the request
	ctxWithTimeout, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Millisecond)
	defer cancel()

	body, err := json.Marshal(input)
	if err != nil {
		return nil, types.NewErrorWithMsg(
			http.StatusInternalServerError,
			types.InternalServiceError,
			"failed to marshal request body",
		)
	}
	req, err := http.NewRequestWithContext(ctxWithTimeout, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, types.NewErrorWithMsg(
			http.StatusInternalServerError, types.InternalServiceError, err.Error(),
		)
	}
	// Set headers
	for key, value := range opts.Headers {
		req.Header.Set(key, value)
	}

	resp, err := client.GetHttpClient().Do(req)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded || err.Error() == "context canceled" {
			return nil, types.NewErrorWithMsg(
				http.StatusRequestTimeout,
				types.RequestTimeout,
				fmt.Sprintf("request timeout after %d ms at %s", timeout, url),
			)
		}
		log.Ctx(ctx).Error().Err(err).Msgf(
			"failed to send request to %s", url,
		)
		return nil, types.NewErrorWithMsg(
			http.StatusInternalServerError,
			types.InternalServiceError,
			fmt.Sprintf("failed to send request to %s", url),
		)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusInternalServerError {
		return nil, types.NewErrorWithMsg(
			resp.StatusCode,
			types.InternalServiceError,
			fmt.Sprintf("internal server error when calling %s", url),
		)
	} else if resp.StatusCode >= http.StatusBadRequest {
		return nil, types.NewErrorWithMsg(
			resp.StatusCode,
			types.BadRequest,
			fmt.Sprintf("client error when calling %s", url),
		)
	}

	var output R
	if err := json.NewDecoder(resp.Body).Decode(&output); err != nil {
		return nil, types.NewErrorWithMsg(
			http.StatusInternalServerError,
			types.InternalServiceError,
			fmt.Sprintf("failed to decode response from %s", url),
		)
	}

	return &output, nil
}
