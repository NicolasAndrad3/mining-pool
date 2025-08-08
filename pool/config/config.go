package config

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Server struct {
		Host string
		Port string
	}
	Database struct {
		URL string
	}
	Security struct {
		APIKey string
	}
	Auth struct {
		Token string
	}
	PoolParams struct {
		MinDifficulty         int
		MaxDifficulty         int
		TargetBlockTime       int
		RewardDistributionCut float64
	}
	Env string
}

func getEnv(key, fallback string) string {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	return val
}

func getEnvAsInt(key string, fallback int) int {
	valStr := os.Getenv(key)
	if valStr == "" {
		return fallback
	}
	val, err := strconv.Atoi(valStr)
	if err != nil {
		log.Fatalf("Erro na conversão de %s: '%s' não é um inteiro válido", key, valStr)
	}
	return val
}

func getEnvAsFloat(key string, fallback float64) float64 {
	valStr := os.Getenv(key)
	if valStr == "" {
		return fallback
	}
	val, err := strconv.ParseFloat(valStr, 64)
	if err != nil {
		log.Fatalf("Erro na conversão de %s: '%s' não é um float válido", key, valStr)
	}
	return val
}

func LoadConfig() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("Aviso: .env não encontrado, usando variáveis do ambiente")
	}

	cfg := &Config{}

	cfg.Server.Host = getEnv("SERVER_HOST", "0.0.0.0")
	cfg.Server.Port = getEnv("SERVER_PORT", "8080")
	cfg.Database.URL = getEnv("DATABASE_URL", "postgres://user:pass@localhost:5432/pool")
	cfg.Security.APIKey = getEnv("API_KEY", "changeme")
	cfg.Auth.Token = getEnv("AUTH_TOKEN", "default-token")
	cfg.Env = getEnv("APP_ENV", "development")

	cfg.PoolParams.MinDifficulty = getEnvAsInt("MIN_DIFFICULTY", 1000)
	cfg.PoolParams.MaxDifficulty = getEnvAsInt("MAX_DIFFICULTY", 100000)
	cfg.PoolParams.TargetBlockTime = getEnvAsInt("TARGET_BLOCK_TIME", 30)
	cfg.PoolParams.RewardDistributionCut = getEnvAsFloat("REWARD_DISTRIBUTION_CUT", 0.02)

	if cfg.PoolParams.RewardDistributionCut < 0 || cfg.PoolParams.RewardDistributionCut > 1 {
		log.Fatalf("REWARD_DISTRIBUTION_CUT inválido: %.2f — deve estar entre 0.0 e 1.0", cfg.PoolParams.RewardDistributionCut)
	}
	if cfg.PoolParams.MinDifficulty >= cfg.PoolParams.MaxDifficulty {
		log.Fatalf("MIN_DIFFICULTY (%d) não pode ser maior ou igual a MAX_DIFFICULTY (%d)", cfg.PoolParams.MinDifficulty, cfg.PoolParams.MaxDifficulty)
	}
	if cfg.PoolParams.TargetBlockTime < 5 {
		log.Fatalf("TARGET_BLOCK_TIME muito baixo: %d segundos — mínimo recomendado é 5s", cfg.PoolParams.TargetBlockTime)
	}

	if cfg.Env == "development" {
		fmt.Println("------ CONFIG DEBUG ------")
		fmt.Printf("%+v\n", *cfg)
		fmt.Println("--------------------------")
	}

	return cfg
}
