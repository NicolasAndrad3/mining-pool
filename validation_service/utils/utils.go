package utils

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
	"unicode"
)

// Retry executa uma função várias vezes com intervalo entre as tentativas.
// Aceita controle de logging e retorna o último erro.
func Retry(attempts int, sleep time.Duration, fn func() error, verbose bool) error {
	var err error
	for i := 1; i <= attempts; i++ {
		err = fn()
		if err == nil {
			return nil
		}
		if verbose {
			fmt.Printf("Tentativa %d falhou: %v\n", i, err)
		}
		time.Sleep(sleep)
	}
	return fmt.Errorf("todas as %d tentativas falharam: %w", attempts, err)
}

// IsPrivateIP verifica se um IP está em faixa privada (IPv4 ou IPv6).
func IsPrivateIP(ip net.IP) bool {
	privateRanges := []net.IPNet{
		{IP: net.IPv4(10, 0, 0, 0), Mask: net.CIDRMask(8, 32)},
		{IP: net.IPv4(172, 16, 0, 0), Mask: net.CIDRMask(12, 32)},
		{IP: net.IPv4(192, 168, 0, 0), Mask: net.CIDRMask(16, 32)},
		{IP: net.ParseIP("fc00::"), Mask: net.CIDRMask(7, 128)},
		{IP: net.ParseIP("fd00::"), Mask: net.CIDRMask(8, 128)},
	}
	for _, r := range privateRanges {
		if r.Contains(ip) {
			return true
		}
	}
	return false
}

// GetClientIP extrai e valida o IP do cabeçalho de requisição.
func GetClientIP(rHeaders map[string]string) (string, error) {
	candidates := []string{
		rHeaders["X-Forwarded-For"],
		rHeaders["X-Real-IP"],
		rHeaders["Remote-Addr"],
	}
	for _, raw := range candidates {
		if raw == "" {
			continue
		}
		ip := strings.TrimSpace(strings.Split(raw, ",")[0])
		parsed := net.ParseIP(ip)
		if parsed != nil {
			return ip, nil
		}
	}
	return "", errors.New("nenhum IP válido encontrado")
}

// ParseDurationSafe converte uma string para duração com fallback em caso de erro.
func ParseDurationSafe(input string, fallback time.Duration) time.Duration {
	d, err := time.ParseDuration(input)
	if err != nil {
		return fallback
	}
	return d
}

// SanitizeInput remove espaços, caracteres inválidos e normaliza campos.
func SanitizeInput(s string) string {
	var builder strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' {
			builder.WriteRune(r)
		}
	}
	return strings.TrimSpace(builder.String())
}

// Contains verifica se um slice contém determinado valor.
// Compatível com tipos básicos e Go 1.18+.
func Contains[T comparable](list []T, target T) bool {
	for _, item := range list {
		if item == target {
			return true
		}
	}
	return false
}
