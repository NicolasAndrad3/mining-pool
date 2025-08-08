package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	c "validation_service/config"
	d "validation_service/database"
	h "validation_service/http"
	l "validation_service/logs"
	m "validation_service/metrics"
	s "validation_service/security"
)

func main() {
	cfg, err := c.Load()
	if err != nil {
		l.Error("config load failed", map[string]interface{}{"err": err})
		os.Exit(1)
	}

	switch cfg.Env {
	case "production", "staging", "development", "test":
	default:
		l.Fatal("invalid environment specified", map[string]interface{}{"env": cfg.Env})
	}

	l.Init("validation_service", cfg.Env, l.LvlInfo, nil)
	l.Info("logger initialized", map[string]interface{}{"env": cfg.Env})

	if err := d.InitializePostgres(cfg.Database.DSN); err != nil {
		l.Fatal("database connection failed", map[string]interface{}{"err": err})
	}
	defer d.ClosePostgres()

	if err := s.InitAuth(cfg.Security); err != nil {
		l.Fatal("security init failed", map[string]interface{}{"err": err})
	}

	if err := m.Setup(cfg.Metrics); err != nil {
		l.Warn("metrics not started", map[string]interface{}{"reason": err.Error()})
	}

	// Simulando dados reais até configuração final dos tokens e origins
	var baseHandler http.Handler = h.NewHandler(map[string]string{}, []string{})

	server := h.NewServer(cfg, baseHandler)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := server.Start(); err != nil {
			l.Fatal("http server crashed", map[string]interface{}{"err": err})
		}
	}()

	<-signalChan
	l.Warn("shutdown signal received", nil)

	shutdownCtx, shutdownCancel := context.WithTimeout(ctx, 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		l.Error("graceful shutdown failed", map[string]interface{}{"err": err})
	} else {
		l.Info("server exited cleanly", nil)
	}
}
