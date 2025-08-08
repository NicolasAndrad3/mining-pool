package types

import (
	"errors"
	"net"
	"regexp"
	"strings"
	"sync"
	"time"
)

type Share struct {
	WorkerID      string    `json:"worker_id"`
	BlockTemplate string    `json:"block_template"`
	Nonce         string    `json:"nonce"`
	Difficulty    int       `json:"difficulty"`
	Timestamp     time.Time `json:"timestamp"`
	IP            net.IP    `json:"ip,omitempty"`
	GPUModel      string    `json:"gpu_model,omitempty"`
	GPUID         string    `json:"gpu_id,omitempty"`
}

const (
	minWorkerIDLen = 6
	maxWorkerIDLen = 64
	maxNonceLen    = 64
	minNonceLen    = 8
	maxShareAge    = 15 * time.Minute
	sep            = "::"
)

var (
	nonceOnce sync.Once
	nonceRe   *regexp.Regexp
)

func getNonceRegex() *regexp.Regexp {
	nonceOnce.Do(func() {
		nonceRe = regexp.MustCompile(`^[a-fA-F0-9]{8,64}$`)
	})
	return nonceRe
}

func (s *Share) Validate() error {
	switch {
	case len(s.WorkerID) < minWorkerIDLen || len(s.WorkerID) > maxWorkerIDLen:
		return errors.New("worker_id inválido")
	case strings.TrimSpace(s.BlockTemplate) == "":
		return errors.New("block_template ausente")
	case !getNonceRegex().MatchString(s.Nonce):
		return errors.New("nonce malformado")
	case s.Difficulty <= 0:
		return errors.New("dificuldade inválida")
	case s.Timestamp.IsZero() || time.Since(s.Timestamp) > maxShareAge:
		return errors.New("timestamp inválido ou muito antigo")
	}
	return nil
}

func (s *Share) Fingerprint() string {
	var b strings.Builder
	b.Grow(len(s.WorkerID) + len(s.Nonce) + len(s.GPUID) + 8)

	b.WriteString(s.WorkerID)
	b.WriteString(sep)
	b.WriteString(s.Nonce)

	if s.GPUID != "" {
		b.WriteString(sep)
		b.WriteString(s.GPUID)
	}
	return b.String()
}
