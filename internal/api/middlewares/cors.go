package middlewares

import (
	"net/http"

	"github.com/babylonchain/staking-api-service/internal/config"
	"github.com/rs/cors"
)

const (
	maxAge                    = 300
	stakerDelegationCheckPath = "/v1/staker/delegation/check"
	galxeOrigin               = "https://app.galxe.com"
)

func CorsMiddleware(cfg *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		// Define a custom CORS policy function
		customCORS := func(r *http.Request) cors.Options {
			// Check if the request path is the special route
			if r.URL.Path == stakerDelegationCheckPath {
				// Return CORS options specific to this route
				return cors.Options{
					AllowedOrigins: []string{galxeOrigin},
					AllowedMethods: []string{"GET", "OPTIONS, POST"},
					MaxAge:         maxAge,
					// Below is a workaround to allow the custom CORS header to be set.
					// i.e OPTIONS will be manually injected into `Access-Control-Allow-Methods` header
					OptionsPassthrough: true,
				}
			}

			// Default CORS options for other routes
			return cors.Options{
				AllowedOrigins: cfg.Server.AllowedOrigins,
				MaxAge:         maxAge,
			}
		}

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Determine CORS options based on the request
			options := customCORS(r)
			// Initialize the CORS handler with the determined options
			cors := cors.New(options)
			corsHandler := cors.Handler(next)

			// Set the custom cors header for the special route for GET requests
			if r.URL.Path == stakerDelegationCheckPath {
				w.Header().Set("Access-Control-Allow-Origin", galxeOrigin)
				w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS, POST")
				if r.Method == http.MethodOptions {
					// This is a preflight request, respond with 204 immediately
					w.WriteHeader(204)
				}
			}
			// Serve the request with the CORS handler
			corsHandler.ServeHTTP(w, r)
		})
	}
}
