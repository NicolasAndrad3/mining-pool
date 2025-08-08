package http

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// WithRequestID garante rastreabilidade da requisição ao injetar um identificador único.
func WithRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = generateDeterministicID()
		}
		w.Header().Set("X-Request-ID", id)
		ctx := context.WithValue(r.Context(), contextKey("request_id"), id)
		ctx = context.WithValue(ctx, contextKey("trace_start"), time.Now())
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// WithStructuredLogging aplica rastreamento contextual da requisição, com status, duração e ID.
func WithStructuredLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capture := &interceptWriter{ResponseWriter: w, code: http.StatusOK}
		start, _ := r.Context().Value(contextKey("trace_start")).(time.Time)
		if start.IsZero() {
			start = time.Now()
		}

		next.ServeHTTP(capture, r)

		duration := time.Since(start)
		reqID, _ := r.Context().Value(contextKey("request_id")).(string)
		if reqID == "" {
			reqID = "-"
		}

		logLine := buildLogLine(r, capture.code, duration, reqID)
		writeAccessLog(logLine)
	})
}

// WithTimeout encerra a requisição se ela exceder o tempo máximo definido.
func WithTimeout(d time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.TimeoutHandler(next, d, "timeout exceeded")
	}
}

// WithSecurityHeaders adiciona headers de defesa contra ameaças comuns (XSS, CSRF, MIME sniffing, etc).
func WithSecurityHeaders(next http.Handler) http.Handler {
	headers := immutableHeaders()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for k, v := range headers {
			w.Header().Set(k, v)
		}
		next.ServeHTTP(w, r)
	})
}

// contextKey define chaves contextuais fortemente tipadas.
type contextKey string

// interceptWriter captura o status code da resposta.
type interceptWriter struct {
	http.ResponseWriter
	code int
}

func (w *interceptWriter) WriteHeader(statusCode int) {
	w.code = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

// writeAccessLog imprime logs da requisição em formato estruturado.
func writeAccessLog(entry string) {
	// Aqui você pode substituir por um logger mais robusto como Zap ou Zerolog futuramente.
	_, _ = io.WriteString(io.Discard, entry) // Troque por stdout ou arquivo
}

// buildLogLine constrói a string de log da requisição.
func buildLogLine(r *http.Request, status int, duration time.Duration, reqID string) string {
	var b strings.Builder
	b.Grow(128)
	b.WriteString("[")
	b.WriteString(r.Method)
	b.WriteString("] ")
	b.WriteString(r.URL.Path)
	b.WriteString(" | ")
	b.WriteString("status=")
	b.WriteString(http.StatusText(status))
	b.WriteString(" code=")
	b.WriteString(strings.TrimPrefix(http.StatusText(status), "HTTP "))
	b.WriteString(" | ")
	b.WriteString("duration=")
	b.WriteString(duration.String())
	b.WriteString(" | req_id=")
	b.WriteString(reqID)
	return b.String()
}

// generateDeterministicID cria um identificador robusto com hash criptográfico.
func generateDeterministicID() string {
	var seed [16]byte
	if _, err := rand.Read(seed[:]); err != nil {
		now := time.Now().UnixNano()
		seed = [16]byte{}
		copy(seed[:], []byte(fmt.Sprint(now)))
	}
	hash := sha256.Sum256(seed[:])
	return hex.EncodeToString(hash[:12]) // 96 bits de entropia são suficientes
}

// immutableHeaders retorna um conjunto de headers de segurança.
func immutableHeaders() map[string]string {
	return map[string]string{
		"X-Content-Type-Options":            "nosniff",
		"X-Frame-Options":                   "DENY",
		"X-XSS-Protection":                  "1; mode=block",
		"Referrer-Policy":                   "strict-origin-when-cross-origin",
		"Permissions-Policy":                "geolocation=(), microphone=(), camera=()",
		"Strict-Transport-Security":         "max-age=31536000; includeSubDomains; preload",
		"Content-Security-Policy":           "default-src 'self'; script-src 'self'; object-src 'none'",
		"X-Permitted-Cross-Domain-Policies": "none",
	}
}

// builderPool opcional (para performance futura com strings.Builder)
var builderPool = sync.Pool{
	New: func() interface{} {
		var b strings.Builder
		b.Grow(128)
		return &b
	},
}
