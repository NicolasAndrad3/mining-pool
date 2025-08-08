package http

import (
	"encoding/json"
	"net/http"
	"time"

	"pool/core"
)

// NewRouter cria um router HTTP com todas as rotas registradas
func NewRouter(pool *core.Pool) http.Handler {
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("/health", withJSON(func(w http.ResponseWriter, r *http.Request) (interface{}, error) {
		return map[string]string{"status": "ok", "timestamp": time.Now().Format(time.RFC3339)}, nil
	}))

	// Rotas principais usando os handlers já implementados
	mux.Handle("/submit", SubmitShareHandler(pool))   // POST
	mux.HandleFunc("/stats", GetPoolStatsHandler)     // GET
	mux.HandleFunc("/test-payout", TestPayoutHandler) // POST

	// (Opcional) Rota legada /shares apenas para placeholder
	mux.HandleFunc("/shares", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "API de shares em construção"})
			return
		}
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
	})

	return mux
}

// --- Helpers para resposta JSON padronizada ---

type handlerFunc func(http.ResponseWriter, *http.Request) (interface{}, error)

func withJSON(fn handlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp, err := fn(w, r)
		if err != nil {
			if httpErr, ok := err.(httpError); ok {
				http.Error(w, httpErr.Message, httpErr.Code)
				return
			}
			http.Error(w, "Erro interno", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}
}

type httpError struct {
	Code    int
	Message string
}

func newHTTPError(code int, msg string) httpError {
	return httpError{Code: code, Message: msg}
}

func (e httpError) Error() string {
	return e.Message
}
