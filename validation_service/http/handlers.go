package http

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"validation_service/core"
	"validation_service/logs"
	"validation_service/security"
	"validation_service/types"
)

type Handler struct {
	validator *core.ValidatorEngine
	auth      *security.TokenVerifier
}

func NewHandler(tokens map[string]string, origins []string) *Handler {
	return &Handler{
		validator: core.NewValidator(),
		auth:      security.NewTokenVerifier(tokens, origins),
	}
}

type shareValidationResponse struct {
	Valid      bool   `json:"valid"`
	Hash       string `json:"hash"`
	LatencyMS  int64  `json:"latency_ms"`
	Suspicious bool   `json:"suspicious"`
	Reason     string `json:"reason,omitempty"`
}

// ServeHTTP implementa a interface http.Handler
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == http.MethodPost && r.URL.Path == "/validate/share":
		h.HandleShare(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (h *Handler) HandleShare(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	reqID := r.Header.Get("X-Request-ID")
	if reqID != "" {
		w.Header().Set("X-Request-ID", reqID)
	}

	if ct := r.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		http.Error(w, "Unsupported Media Type", http.StatusUnsupportedMediaType)
		logs.Warn("Rejected share: invalid content-type", nil)
		return
	}

	authMeta, err := h.auth.AuthenticateRequest(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		logs.Warn("Rejected share: "+err.Error(), nil)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	defer r.Body.Close()

	var share types.Share
	if decodeErr := json.NewDecoder(r.Body).Decode(&share); decodeErr != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		logs.Error("Erro ao decodificar JSON do share", map[string]interface{}{"error": decodeErr})
		return
	}

	validationResult := h.validator.ValidateShare(share)

	response := shareValidationResponse{
		Valid:      validationResult.IsValid,
		Hash:       validationResult.ComputedHash,
		LatencyMS:  validationResult.LatencyMS,
		Suspicious: validationResult.Suspicious,
		Reason:     validationResult.ErrorReason,
	}

	logs.Info("Share validado", map[string]interface{}{
		"worker_id":  share.WorkerID,
		"latency_ms": validationResult.LatencyMS,
		"valid":      validationResult.IsValid,
		"suspicious": validationResult.Suspicious,
		"ip":         authMeta.IP,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)

	_ = time.Since(start) // Para métricas futuras
}
