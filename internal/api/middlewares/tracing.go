package middlewares

import (
	"context"
	"net/http"

	"github.com/babylonchain/staking-api-service/internal/observability/tracing"
	"github.com/google/uuid"
)

func TracingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Attach traceId into request context
		traceID := uuid.New().String()
		ctx := context.WithValue(r.Context(), tracing.TraceIdKey, traceID)

		// Start tracingInfo
		tracingInfo := &tracing.TracingInfo{}
		ctx = context.WithValue(ctx, tracing.TracingInfoKey, tracingInfo)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
