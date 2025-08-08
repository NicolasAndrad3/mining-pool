package security

import "validation_service/logs"

// InitAuth é responsável por inicializar subsistemas de segurança, como preload de listas, keys, certificados, etc.
func InitAuth(cfg interface{}) error {
	logs.Info("segurança inicializada", nil)
	// Futuramente pode verificar certificados, JWTs, etc.
	return nil
}
