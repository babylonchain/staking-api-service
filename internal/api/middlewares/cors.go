package middlewares

import (
	"net/http"

	"github.com/babylonchain/staking-api-service/internal/config"
	"github.com/rs/cors"
)

const (
	maxAge = 300
)

func CorsMiddleware(cfg *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Determine CORS options based on the request
			options := cors.Options{
				AllowedOrigins: cfg.Server.AllowedOrigins,
				MaxAge:         maxAge,
			}
			// Initialize the CORS handler with the determined options
			cors := cors.New(options)
			corsHandler := cors.Handler(next)

			// Serve the request with the CORS handler
			corsHandler.ServeHTTP(w, r)
		})
	}
}
