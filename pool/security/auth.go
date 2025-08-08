package security

import (
	"net/http"
	"strings"

	"pool/logs"
	"pool/utils"
)

type Authenticator struct {
	apiKey string
}

func NewAuthenticator(apiKey string) *Authenticator {
	return &Authenticator{apiKey: apiKey}
}

func (a *Authenticator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := utils.GetRequestID(r.Context())

		token := extractToken(r, requestID)
		if token == "" {
			logs.WithFields(map[string]interface{}{
				"request_id": requestID,
				"remote":     r.RemoteAddr,
			}).Warn("[auth] No token provided")
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		if token != a.apiKey {
			logs.WithFields(map[string]interface{}{
				"request_id": requestID,
				"token":      token,
			}).Warn("[auth] Invalid token")
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func extractToken(r *http.Request, requestID string) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		logs.WithFields(map[string]interface{}{
			"request_id": requestID,
			"headers":    r.Header,
		}).Warn("[auth] Authorization header missing")
		return ""
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		logs.WithFields(map[string]interface{}{
			"request_id": requestID,
			"authHeader": authHeader,
		}).Warn("[auth] Malformed Authorization header")
		return ""
	}

	return parts[1]
}
