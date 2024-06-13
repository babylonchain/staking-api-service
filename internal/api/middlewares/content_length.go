package middlewares

import (
	"net/http"

	"github.com/babylonchain/staking-api-service/internal/config"
)


func ContentLengthMiddleware(cfg *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost {
				if r.ContentLength > int64(cfg.Server.MaxContentLength) {
					http.Error(w, "Request Entity Too Large", http.StatusRequestEntityTooLarge)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}