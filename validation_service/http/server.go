package http

import (
	"context"
	"net"
	"net/http"
	"time"

	"validation_service/config"

	"validation_service/logs"
)

type Server struct {
	httpServer *http.Server
	cfg        *config.Config
	handler    http.Handler
}

// NewServer aplica middlewares à cadeia de handlers e retorna um servidor configurado
func NewServer(cfg *config.Config, baseHandler http.Handler) *Server {
	middlewareStack := chainMiddlewares(
		WithRequestID,
		WithStructuredLogging,
		WithTimeout(12*time.Second),
		WithSecurityHeaders,
	)

	enhancedHandler := middlewareStack(baseHandler)

	return &Server{
		cfg:     cfg,
		handler: enhancedHandler,
	}
}

// Start inicia o servidor HTTP e registra logs relevantes
func (s *Server) Start() error {
	address := net.JoinHostPort(s.cfg.Server.Host, s.cfg.Server.Port)

	s.httpServer = &http.Server{
		Addr:              address,
		Handler:           s.handler,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	logs.WithFields(map[string]interface{}{
		"env":  s.cfg.Env,
		"host": s.cfg.Server.Host,
		"port": s.cfg.Server.Port,
	}).Info("Validation server initialized")

	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logs.WithError(err).Fatal("Critical failure during server start")
		return err
	}

	return nil
}

// Shutdown finaliza o servidor de forma controlada
func (s *Server) Shutdown(ctx context.Context) error {
	logs.Warn("Shutting down validation server gracefully")
	return s.httpServer.Shutdown(ctx)
}

// chainMiddlewares aplica middlewares em cadeia na ordem inversa
func chainMiddlewares(mw ...func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(final http.Handler) http.Handler {
		for i := len(mw) - 1; i >= 0; i-- {
			final = mw[i](final)
		}
		return final
	}
}
