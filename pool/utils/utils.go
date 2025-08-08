package utils

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"net"
	"regexp"
	"strings"
	"time"

	"pool/logs"
)

type RetryableFunc func() error

func Retry(attempts int, delay time.Duration, fn RetryableFunc) error {
	var err error
	for i := 0; i < attempts; i++ {
		err = fn()
		if err == nil {
			return nil
		}
		logs.Debugf("[RETRY] Tentativa %d falhou: %v. Retentando em %s", i+1, err, delay)
		time.Sleep(delay)
	}
	return fmt.Errorf("todas as %d tentativas falharam: %w", attempts, err)
}

type TimeoutFunc func(ctx context.Context) error

// Timeout executa função com deadline
func Timeout(fn TimeoutFunc, limit time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), limit)
	defer cancel()

	err := fn(ctx)
	if err != nil {
		return fmt.Errorf("erro ao executar com timeout: %w", err)
	}
	return nil
}

func TimeNowUTC() string {
	now := time.Now().UTC()
	// fallback de sanity
	if now.Before(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)) {
		return "1970-01-01T00:00:00Z"
	}
	return now.Format(time.RFC3339)
}

func RandomString(length int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	bound := big.NewInt(int64(len(charset)))

	var sb strings.Builder
	sb.Grow(length)

	for i := 0; i < length; i++ {
		num, err := rand.Int(rand.Reader, bound)
		if err != nil {
			return "", fmt.Errorf("erro gerando número aleatório: %w", err)
		}
		sb.WriteByte(charset[num.Int64()])
	}

	return sb.String(), nil
}

func RandomHex(n int) (string, error) {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("falha ao gerar bytes aleatórios: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

func CleanInput(input string) string {
	re := regexp.MustCompile(`[^\pL\pN_\-@.\s]`)
	clean := re.ReplaceAllString(strings.TrimSpace(input), "")
	return clean
}

var (
	ErrInvalidCIDR     = errors.New("CIDR inválido")
	ErrIPOutsideSubnet = errors.New("IP fora da faixa do CIDR")
)

func IsIPAllowed(ipStr string, cidr string) (bool, error) {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false, fmt.Errorf("IP inválido: %s", ipStr)
	}

	_, subnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return false, fmt.Errorf("%w: %s", ErrInvalidCIDR, cidr)
	}

	if subnet.Contains(ip) {
		return true, nil
	}

	return false, fmt.Errorf("%w: %s não está dentro de %s", ErrIPOutsideSubnet, ipStr, cidr)
}
