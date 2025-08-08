package main

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"pool/config"
	phttp "pool/http"
	"pool/logs"
	"pool/security"
	"pool/smartcontract"
)

func printAsciiBanner() {
	banner := `                
 ____   ___  ____    _____ ____   ____  ____       ____   ___    ___   _     
|    \ /  _]|    \  / ___/|    | /    ||    \     |    \ /   \  /   \ | |    
|  o  )  [_ |  D  )(   \_  |  | |  o  ||  _  |    |  o  )     ||     || |    
|   _/    _]|    /  \__  | |  | |     ||  |  |    |   _/|  O  ||  O  || |___ 
|  | |   [_ |    \  /  \ | |  | |  _  ||  |  |    |  |  |     ||     ||     |
|  | |     ||  .  \ \    | |  | |  |  ||  |  |    |  |  |     ||     ||     |
|__| |_____||__|\_|  \___||____||__|__||__|__|    |__|   \___/  \___/ |_____| 

                      /^--^\     /^--^\     /^--^\
                      \____/     \____/     \____/
                     /      \   /      \   /      \
                    |        | |        | |        |
                     \__  __/   \__  __/   \__  __/
|^|^|^|^|^|^|^|^|^|^|^|^\ \^|^|^|^/ /^|^|^|^|^\ \^|^|^|^|^|^|^|^|^|^|^|^|
| | | | | | | | | | | | |\ \| | |/ /| | | | | | \ \ | | | | | | | | | | |
########################/ /######\ \###########/ /#######################
| | | | | | | | | | | | \/| | | | \/| | | | | |\/ | | | | | | | | | | | |
|_|_|_|_|_|_|_|_|_|_|_|_|_|_|_|_|_|_|_|_|_|_|_|_|_|_|_|_|_|_|_|_|_|_|_|_|

`
	fmt.Println(banner)
}

func main() {
	printAsciiBanner()

	cfg := config.LoadConfig()
	if cfg.Server.Port == "" || cfg.Database.URL == "" {
		logs.WithFields(map[string]interface{}{
			"missing_fields": []string{"SERVER_PORT", "DATABASE_URL"},
		}).Fatal("Missing critical configuration")
	}

	env := cfg.Env
	if env == "" {
		if e := os.Getenv("ENV"); e != "" {
			env = e
		} else {
			env = "development"
		}
	}
	logs.Init(env)
	defer logs.CloseLogFile()

	start := time.Now()
	logs.WithFields(map[string]interface{}{
		"timestamp": start.Format(time.RFC3339),
		"env":       env,
	}).Info("Starting pool server initialization...")

	// Removed logging the full config to avoid leaking sensitive info

	if err := security.LoadSecrets(cfg); err != nil {
		logs.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Fatal("Failed to initialize security module")
	}

	// Initialization of database connection, shareStore and smartcontract
	// should be done securely without exposing sensitive data in logs.

	// Example placeholder for connecting and using shareStore:
	// db, err := database.Connect(cfg.Database.URL)
	// if err != nil {
	//	   logs.WithFields(map[string]interface{}{"error": err.Error()}).Fatal("Database connection failed")
	// }
	// defer db.Close()
	// shareStore := database.NewPostgresShareStore(db)

	paymentEngine, err := smartcontract.Init(cfg)
	if err != nil {
		logs.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Fatal("Smart contract initialization failed")
	}

	if rs, ok := paymentEngine.(interface {
		SendReward(string, *big.Int) (string, error)
	}); ok {
		phttp.SetPaymentClient(rs)
	} else {
		logs.Warn("Payment engine doesn't expose SendReward; /test-payout will be limited")
	}

	// You can adjust pool creation according to your real implementation
	// pool := core.NewPool(cfg, db, paymentEngine, shareStore)

	// Start pool logic if available, e.g., pool.Start(ctx)

	router := phttp.NewRouter(nil) // Pass your real pool instance here
	server := phttp.NewServer(cfg, router)

	errChan := make(chan error, 1)
	go func() {
		errChan <- server.Start()
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	select {
	case <-sig:
		logs.Warn("Termination signal received. Initiating shutdown sequence...")
	case err := <-errChan:
		if err != nil {
			logs.WithFields(map[string]interface{}{
				"error": err.Error(),
			}).Error("HTTP server crashed unexpectedly")
		}
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logs.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Error("Graceful shutdown failed")
	} else {
		logs.Info("Server shutdown completed successfully")
	}

	logs.WithFields(map[string]interface{}{
		"uptime": time.Since(start).String(),
	}).Info("Shutdown complete")
}

func redact(s string) string {
	if s == "" {
		return ""
	}
	if len(s) <= 6 {
		return "***"
	}
	return s[:3] + "****" + s[len(s)-3:]
}

func redactDSN(dsn string) string {
	at := ""
	if i := strings.Index(dsn, "@"); i > -1 {
		if j := strings.Index(dsn, "://"); j > -1 && i > j {
			at = dsn[i:]
			return dsn[:j+3] + "***" + at
		}
	}
	return dsn
}
