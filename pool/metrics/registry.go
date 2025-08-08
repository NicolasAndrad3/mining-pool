package metrics

import (
	"net/http"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	registry     *prometheus.Registry
	registryOnce sync.Once
)

func InitRegistry() {
	registryOnce.Do(func() {
		registry = prometheus.NewRegistry()

		registry.MustRegister(prometheus.NewGoCollector())
		registry.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))

		registry.MustRegister(SharesValid)
		registry.MustRegister(SharesInvalid)
		registry.MustRegister(ValidationDuration)
		registry.MustRegister(JobsActive)
		registry.MustRegister(WorkersConnected)
	})
}

func Handler() http.Handler {
	if registry == nil {
		InitRegistry()
	}
	return promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
}
