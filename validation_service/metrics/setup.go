package metrics

import "validation_service/logs"

// Setup inicializa coleta de métricas (Prometheus, OpenTelemetry, etc)
func Setup(cfg interface{}) error {
	logs.Info("coleta de métricas configurada", nil)
	// Pode futuramente inicializar exporsers, goroutines, etc.
	return nil
}
