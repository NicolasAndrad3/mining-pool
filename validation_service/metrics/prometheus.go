package metrics

import (
	"net/http"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	reqCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "validator",
			Subsystem: "http",
			Name:      "requests_total",
			Help:      "Total number of HTTP requests received",
		},
		[]string{"method", "route", "code"},
	)

	reqDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "validator",
			Subsystem: "http",
			Name:      "request_duration_seconds",
			Help:      "Duration of HTTP requests in seconds",
			Buckets:   prometheus.ExponentialBuckets(0.005, 2, 12),
		},
		[]string{"method", "route"},
	)
)

// registerCoreMetrics ensures idempotent registration via init
func init() {
	prometheus.MustRegister(reqCount, reqDuration)
}

// InstrumentHandler wraps an HTTP handler and records Prometheus metrics.
func InstrumentHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sanitizedPath := routeLabelSanitizer(r.URL.Path)
		start := time.Now()

		rw := &respWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(rw, r)

		elapsed := time.Since(start).Seconds()

		reqCount.WithLabelValues(r.Method, sanitizedPath, http.StatusText(rw.statusCode)).Inc()
		reqDuration.WithLabelValues(r.Method, sanitizedPath).Observe(elapsed)
	})
}

// ExposeMetricsEndpoint binds the /metrics route with proper timeout protection
func ExposeMetricsEndpoint(mux *http.ServeMux) {
	mux.Handle("/metrics", http.TimeoutHandler(
		promhttp.Handler(),
		3*time.Second,
		`metrics timeout`,
	))
}

// respWriter helps capture HTTP status codes
type respWriter struct {
	http.ResponseWriter
	statusCode int
}

func (r *respWriter) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

// routeLabelSanitizer strips dynamic parts from URL paths to reduce label cardinality
func routeLabelSanitizer(path string) string {
	segments := strings.Split(path, "/")
	for i, segment := range segments {
		if len(segment) > 12 || isHexLike(segment) {
			segments[i] = ":id"
		}
	}
	return strings.Join(segments, "/")
}

// isHexLike detects hex-style strings often used in IDs or hashes
func isHexLike(str string) bool {
	if len(str) < 6 {
		return false
	}
	count := 0
	for _, ch := range str {
		if (ch >= 'a' && ch <= 'f') || (ch >= '0' && ch <= '9') {
			count++
		}
	}
	return count > len(str)/2
}
