package core

import (
	"crypto/sha256"
	"encoding/hex"
	"math/big"
	"os"
	"strings"
	"time"

	"validation_service/logs"
	"validation_service/types"
)

// ValidationResult representa o resultado completo da validação de um share.
type ValidationResult struct {
	IsValid      bool
	ErrorReason  string
	ComputedHash string
	LatencyMS    int64
	Suspicious   bool
}

// ValidatorEngine executa validações técnicas nos shares recebidos.
type ValidatorEngine struct {
	MinLatencyMS int64  // Latência mínima esperada entre shares
	TargetHex    string // Target hexadecimal (simulando blockchains reais)
	DevMode      bool   // Modo desenvolvimento: validação leniente
}

// NewValidator cria uma instância do validador.
// Em DEV (ENV=development), ativa modo leniente.
func NewValidator() *ValidatorEngine {
	dev := strings.EqualFold(os.Getenv("ENV"), "development")
	return &ValidatorEngine{
		MinLatencyMS: 100,
		TargetHex:    "00000fffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		DevMode:      dev,
	}
}

// ValidateShare realiza a validação técnica de um share.
func (v *ValidatorEngine) ValidateShare(share types.Share) ValidationResult {
	start := time.Now()

	// Validação preliminar do share (campo obrigatório, nonce, etc)
	if err := share.Validate(); err != nil {
		return ValidationResult{
			IsValid:      false,
			ErrorReason:  err.Error(),
			ComputedHash: "",
			LatencyMS:    time.Since(start).Milliseconds(),
			Suspicious:   true,
		}
	}

	hash := v.computeHash(share.BlockTemplate, share.Nonce)
	latency := time.Since(start).Milliseconds()

	// --- DEV MODE: aceitar formato/shape mínimo para destravar integração ---
	if v.DevMode {
		if isHex256(hash) {
			res := ValidationResult{
				IsValid:      true,
				ErrorReason:  "",
				ComputedHash: hash,
				LatencyMS:    latency,
				Suspicious:   latency < v.MinLatencyMS,
			}
			v.logResult(share, res)
			return res
		}
		// se por algum motivo não for hex-256, cai para as validações padrão para erro claro
	}

	valid := v.hashBelowTarget(hash)
	result := ValidationResult{
		IsValid:      valid,
		ErrorReason:  "",
		ComputedHash: hash,
		LatencyMS:    latency,
		Suspicious:   latency < v.MinLatencyMS,
	}

	if !valid {
		result.ErrorReason = "hash acima do target"
	}

	v.logResult(share, result)
	return result
}

// computeHash concatena dados e retorna o hash em hexadecimal.
func (v *ValidatorEngine) computeHash(data, nonce string) string {
	raw := data + nonce
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// hashBelowTarget verifica se o hash gerado está abaixo do target.
func (v *ValidatorEngine) hashBelowTarget(hash string) bool {
	if !isHex256(hash) {
		return false
	}
	h := new(big.Int)
	t := new(big.Int)
	h.SetString(hash, 16)
	t.SetString(v.TargetHex, 16)
	return h.Cmp(t) == -1
}

func isHex256(s string) bool {
	if len(s) != 64 {
		return false
	}
	_, err := hex.DecodeString(s)
	return err == nil
}

// logResult registra o resultado técnico do share.
func (v *ValidatorEngine) logResult(share types.Share, res ValidationResult) {
	logs.Infof("VALIDATION: worker=%s hash=%s latency=%dms valid=%t suspicious=%t dev=%t",
		share.WorkerID, truncate(res.ComputedHash, 12), res.LatencyMS, res.IsValid, res.Suspicious, v.DevMode)

	if res.ErrorReason != "" {
		logs.Warnf("REJECTED: %s (%s)", share.WorkerID, res.ErrorReason)
	}
}

// truncate retorna apenas os primeiros n caracteres do hash.
func truncate(s string, n int) string {
	if len(s) < n {
		return s
	}
	return s[:n]
}
