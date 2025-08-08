package http

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"pool/logs"
	"pool/utils"
)

// Assinatura de middleware
type Middleware func(http.Handler) http.Handler

// Aplica a cadeia de middlewares
func ApplyMiddleware(h http.Handler, chain ...Middleware) http.Handler {
	for i := len(chain) - 1; i >= 0; i-- {
		h = chain[i](h)
	}
	return h
}

// Middleware: adiciona um ID único por requisição
func withRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := fmt.Sprintf("req-%d", time.Now().UnixNano())
		ctx := utils.SetRequestID(r.Context(), id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Middleware: adiciona timeout por requisição
func withTimeout(timeout time.Duration) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// Middleware: cabeçalhos de segurança padrão
func withSecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-Xss-Protection", "1; mode=block")
		next.ServeHTTP(w, r)
	})
}

// Middleware: log estruturado
func withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		requestID := utils.GetRequestID(r.Context())
		next.ServeHTTP(w, r)
		duration := time.Since(start)

		logs.WithFields(map[string]interface{}{
			"method":     r.Method,
			"path":       r.URL.Path,
			"remote":     r.RemoteAddr,
			"duration":   duration.String(),
			"request_id": requestID,
		}).Info("Request completed")
	})
}

// Middleware: CORS
// - Se origins contiver "*", libera geral com "Access-Control-Allow-Origin: *"
// - Caso contrário, reflete o Origin somente se ele estiver na lista
func withCORS(origins []string) Middleware {
	allowAll := false
	for _, o := range origins {
		if o == "*" {
			allowAll = true
			break
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Evitar problemas de cache intermediário
			w.Header().Add("Vary", "Origin")
			w.Header().Add("Vary", "Access-Control-Request-Method")
			w.Header().Add("Vary", "Access-Control-Request-Headers")

			if allowAll {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			} else if origin != "" {
				for _, o := range origins {
					if o == origin {
						w.Header().Set("Access-Control-Allow-Origin", origin)
						break
					}
				}
			}

			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			// opcional: cache do preflight
			w.Header().Set("Access-Control-Max-Age", "600")

			// Preflight
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// Middleware: verificação de token (libera /health e /metrics e OPTIONS)
func withAuthToken(expectedToken string) Middleware {
	skipAuth := map[string]bool{
		"/health":  true,
		"/metrics": true,
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Libera endpoints públicos e preflight
			if r.Method == http.MethodOptions || skipAuth[r.URL.Path] {
				next.ServeHTTP(w, r)
				return
			}

			const prefix = "Bearer "
			authHeader := r.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, prefix) {
				logs.WithFields(map[string]interface{}{
					"remote": r.RemoteAddr,
					"path":   r.URL.Path,
				}).Warn("Unauthorized access attempt (missing bearer)")
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			token := strings.TrimPrefix(authHeader, prefix)
			if token != expectedToken {
				logs.WithFields(map[string]interface{}{
					"remote": r.RemoteAddr,
					"path":   r.URL.Path,
				}).Warn("Unauthorized access attempt (invalid token)")
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
