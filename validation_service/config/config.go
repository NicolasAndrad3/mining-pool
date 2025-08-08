package config

import (
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Env      string
	Port     string
	Database Database
	Metrics  Metrics
	Security Security
	Server   ServerConfig // <- Adicionado aqui
}

type Database struct {
	DSN         string
	MaxIdle     int
	MaxOpen     int
	ConnTimeout time.Duration
}

type Metrics struct {
	Enabled bool
	Port    string
}

type Security struct {
	APIKey      string
	AllowedCIDR string
	JWTSecret   string
}

type ServerConfig struct {
	Host string
	Port string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		Env:  getenv("APP_ENV", detectEnv()),
		Port: getenv("PORT", "8081"),
		Database: Database{
			DSN:         getenv("DATABASE_URL", ""),
			MaxIdle:     atoi(getenv("DB_MAX_IDLE", "10")),
			MaxOpen:     atoi(getenv("DB_MAX_OPEN", "100")),
			ConnTimeout: parseDuration(getenv("DB_TIMEOUT", "10s")),
		},
		Metrics: Metrics{
			Enabled: getbool(getenv("METRICS_ENABLED", "true")),
			Port:    getenv("METRICS_PORT", "9090"),
		},
		Security: Security{
			APIKey:      getenv("API_KEY", ""),
			AllowedCIDR: getenv("ALLOWED_CIDR", "0.0.0.0/0"),
			JWTSecret:   getenv("JWT_SECRET", "default_jwt_secret_123"),
		},
		Server: ServerConfig{ // <- Novo bloco
			Host: getenv("SERVER_HOST", "0.0.0.0"),
			Port: getenv("SERVER_PORT", "8081"),
		},
	}

	warnings, err := validate(cfg)
	for _, w := range warnings {
		log.Println("[config warning]", w)
	}
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

func detectEnv() string {
	h, err := os.Hostname()
	if err != nil {
		return "development"
	}
	switch {
	case strings.Contains(h, "staging"):
		return "staging"
	case strings.Contains(h, "prod"):
		return "production"
	default:
		return "development"
	}
}

func getenv(k, fallback string) string {
	v := strings.TrimSpace(os.Getenv(k))
	if v == "" {
		return fallback
	}
	return v
}

func atoi(val string) int {
	n, _ := strconv.Atoi(val)
	return n
}

func getbool(v string) bool {
	b, err := strconv.ParseBool(v)
	if err != nil {
		return false
	}
	return b
}

func parseDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 10 * time.Second
	}
	return d
}

func validate(cfg *Config) ([]string, error) {
	var warn []string

	if cfg.Database.DSN == "" {
		warn = append(warn, "DATABASE_URL not set — some services may fail")
	}

	if _, _, err := net.ParseCIDR(cfg.Security.AllowedCIDR); err != nil {
		return warn, fmt.Errorf("invalid CIDR: %v", err)
	}

	if cfg.Security.APIKey == "" || len(cfg.Security.APIKey) < 12 {
		return warn, fmt.Errorf("invalid or missing API_KEY")
	}

	if _, err := strconv.Atoi(cfg.Port); err != nil {
		return warn, fmt.Errorf("invalid port format: %v", err)
	}

	return warn, nil
}

func (c *Config) SafeLog() {
	masked := "********"
	fmt.Println("[config] ENV:", c.Env)
	fmt.Println("[config] PORT:", c.Port)
	fmt.Println("[config] DB:", mask(c.Database.DSN))
	fmt.Println("[config] CIDR:", c.Security.AllowedCIDR)
	fmt.Println("[config] API_KEY:", masked)
	fmt.Println("[config] JWT_SECRET:", masked)
}

func mask(input string) string {
	if len(input) < 5 {
		return "*****"
	}
	return input[:2] + strings.Repeat("*", len(input)-4) + input[len(input)-2:]
}
