package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	SharesValid = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "pool_shares_valid_total",
		Help: "Total de shares válidos processados",
	})
	SharesInvalid = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "pool_shares_invalid_total",
		Help: "Total de shares inválidos processados",
	})

	ValidationDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "pool_share_validation_seconds",
		Help:    "Tempo de validação de shares em segundos",
		Buckets: prometheus.DefBuckets,
	})

	JobsActive = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "pool_jobs_active",
		Help: "Número de jobs ativos no momento",
	})

	WorkersConnected = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "pool_workers_connected",
		Help: "Número de workers conectados à pool",
	})

	InternalErrors = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "pool_internal_errors_total",
		Help: "Total de erros internos no processamento",
	})
)
