package http

import (
	"context"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"pool/config"
	"pool/logs"
	"pool/metrics"
)

// Server estrutura o servidor HTTP
type Server struct {
	engine         *http.Server
	cfg            *config.Config
	handler        http.Handler
	allowedOrigins []string
}

// NewServer aplica os middlewares e monta o servidor
func NewServer(cfg *config.Config, router http.Handler) *Server {
	// Inicializa métricas Prometheus
	metrics.InitRegistry()

	// Cria um mux que combina as rotas normais com /metrics
	mux := http.NewServeMux()
	mux.Handle("/", router) // rotas principais da pool
	mux.Handle("/metrics", metrics.Handler())

	// Carrega origens permitidas de ENV ou config
	allowedOrigins := []string{"https://seu-dominio.com"}
	if envVal := os.Getenv("POOL_ALLOWED_ORIGINS"); envVal != "" {
		allowedOrigins = strings.Split(envVal, ",")
	}

	// Cadeia de middlewares
	middlewares := []Middleware{
		withRequestID,
		withSecurityHeaders,
		withCORS(allowedOrigins),
		withAuthToken(cfg.Security.APIKey),
		withTimeout(10 * time.Second),
		withLogging,
	}

	// Aplicar cadeia
	finalHandler := ApplyMiddleware(mux, middlewares...)

	return &Server{
		cfg:            cfg,
		handler:        finalHandler,
		allowedOrigins: allowedOrigins,
	}
}

// Start inicia o servidor HTTP com parâmetros avançados
func (s *Server) Start() error {
	s.engine = &http.Server{
		Addr:              net.JoinHostPort(s.cfg.Server.Host, s.cfg.Server.Port),
		Handler:           s.handler,
		ReadTimeout:       20 * time.Second,
		ReadHeaderTimeout: 8 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1MB
	}

	logs.WithFields(map[string]interface{}{
		"host":            s.cfg.Server.Host,
		"port":            s.cfg.Server.Port,
		"allowed_origins": strings.Join(s.allowedOrigins, ","),
	}).Info("HTTP server starting...")

	return s.engine.ListenAndServe()
}

// Shutdown finaliza o servidor com grace period
func (s *Server) Shutdown(ctx context.Context) error {
	logs.Warn("Gracefully shutting down HTTP server...")

	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := s.engine.Shutdown(shutdownCtx); err != nil {
		logs.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Error("Shutdown failed")
		return err
	}

	logs.Info("HTTP server shut down cleanly.")
	return nil
}
