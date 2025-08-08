package security

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"validation_service/logs"
)

// AuthMetadata contém os metadados extraídos da requisição autenticada.
type AuthMetadata struct {
	Token     string
	Role      string
	UserAgent string
	IP        string
	Timestamp time.Time
}

// TokenVerifier realiza a verificação de tokens e origens permitidas.
type TokenVerifier struct {
	ValidTokens    map[string]string // Exemplo: map[token]role
	AllowedOrigins []string          // Exemplo: ["https://meusite.com"]
}

// NewTokenVerifier cria um verificador configurado.
func NewTokenVerifier(tokens map[string]string, origins []string) *TokenVerifier {
	return &TokenVerifier{
		ValidTokens:    tokens,
		AllowedOrigins: origins,
	}
}

// AuthenticateRequest valida o token e a origem.
func (tv *TokenVerifier) AuthenticateRequest(r *http.Request) (*AuthMetadata, error) {
	auth := r.Header.Get("Authorization")
	token := strings.TrimPrefix(auth, "Bearer ")

	if len(token) == 0 || len(token) > 512 {
		return nil, errors.New("token ausente ou inválido")
	}

	role, ok := tv.ValidTokens[token]
	if !ok {
		logs.Warn("TOKEN INVÁLIDO", map[string]interface{}{
			"ip":     extractClientIP(r),
			"ua":     r.UserAgent(),
			"origin": r.Header.Get("Origin"),
		})
		return nil, errors.New("acesso negado")
	}

	origin := r.Header.Get("Origin")
	if !tv.isAllowedOrigin(origin) {
		logs.Warn("ORIGEM BLOQUEADA", map[string]interface{}{
			"origin": origin,
		})
		return nil, errors.New("origem não permitida")
	}

	if r.TLS == nil {
		logs.Warn("REQUISIÇÃO SEM TLS", map[string]interface{}{
			"ip": extractClientIP(r),
			"ua": r.UserAgent(),
		})
		// return nil, errors.New("TLS obrigatório")
	}

	return &AuthMetadata{
		Token:     token,
		Role:      role,
		UserAgent: r.UserAgent(),
		IP:        extractClientIP(r),
		Timestamp: time.Now(),
	}, nil
}

// isAllowedOrigin verifica se a origem está autorizada.
func (tv *TokenVerifier) isAllowedOrigin(origin string) bool {
	origin = strings.ToLower(origin)
	for _, allowed := range tv.AllowedOrigins {
		if strings.Contains(origin, allowed) {
			return true
		}
	}
	return false
}

// extractClientIP retorna o IP real do cliente.
func extractClientIP(r *http.Request) string {
	if ip := r.Header.Get("CF-Connecting-IP"); ip != "" {
		return ip
	}
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return strings.Split(ip, ",")[0]
	}
	ip, _, _ := strings.Cut(r.RemoteAddr, ":")
	return ip
}
