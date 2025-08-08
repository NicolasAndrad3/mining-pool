package security

import (
	"errors"
	"pool/config"
)

func LoadSecrets(cfg *config.Config) error {
	if cfg.Security.APIKey == "" {
		return errors.New("API key not configured")
	}
	// Futuramente: carregar certificados, etc.
	return nil
}
