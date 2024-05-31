package middlewares

import (
	"net/http"

	"github.com/rs/zerolog/log"
	"github.com/unrolled/secure"
)

// SecurityHeadersMiddleware sets various security headers using the unrolled/secure package
func SecurityHeadersMiddleware() func(http.Handler) http.Handler {
	sec := secure.New(secure.Options{
		FrameDeny:             true, // Equivalent to X-Frame-Options: DENY
		ContentTypeNosniff:    true, // Equivalent to X-Content-Type-Options: nosniff
		BrowserXssFilter:      true, // Equivalent to X-XSS-Protection: 1; mode=block
		ContentSecurityPolicy: `
			default-src 'self'; 
			script-src 'self' 'unsafe-inline' https://cdnjs.cloudflare.com https://stackpath.bootstrapcdn.com;
			style-src 'self' 'unsafe-inline' https://cdnjs.cloudflare.com https://stackpath.bootstrapcdn.com;
			img-src 'self' data: https://cdnjs.cloudflare.com https://stackpath.bootstrapcdn.com;
			font-src 'self' https://cdnjs.cloudflare.com https://stackpath.bootstrapcdn.com;
			object-src 'none';
			frame-ancestors 'self';
			form-action 'self';
			block-all-mixed-content;
			base-uri 'self';
		`,
		ReferrerPolicy: "strict-origin-when-cross-origin", // Setting Referrer-Policy
	})

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Apply the secure middleware
			err := sec.Process(w, r)
			if err != nil {
				// If there's an error, do not proceed further
				log.Error().Err(err).Msg("error while applying security headers")
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			// Call the next handler
			next.ServeHTTP(w, r)
		})
	}
}