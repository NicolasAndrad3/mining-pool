package database

import (
	"database/sql"
	"fmt"
	"time"

	"validation_service/logs"

	_ "github.com/lib/pq"
)

var DB *sql.DB

func InitializePostgres(connectionString string) error {
	logs.Info("Iniciando conexão com PostgreSQL", map[string]interface{}{
		"dsn": connectionString,
	})
	fmt.Println("📡 Tentando conectar ao PostgreSQL com DSN:", connectionString)

	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		fmt.Println("🔴 Erro ao abrir conexão:", err)
		return fmt.Errorf("erro ao abrir conexão com postgres: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	if pingErr := db.Ping(); pingErr != nil {
		fmt.Println("🔴 Erro no Ping():", pingErr)
		return fmt.Errorf("não foi possível conectar ao banco de dados: %w", pingErr)
	}

	createTableQuery := `
	CREATE TABLE IF NOT EXISTS shares (
		id SERIAL PRIMARY KEY,
		worker_id VARCHAR(255) NOT NULL,
		share_hash VARCHAR(255) NOT NULL UNIQUE,
		valid BOOLEAN NOT NULL,
		timestamp TIMESTAMP NOT NULL DEFAULT NOW(),
		difficulty INTEGER
	);`

	fmt.Println("🛠️ Executando criação da tabela shares...")

	if _, execErr := db.Exec(createTableQuery); execErr != nil {
		fmt.Println("🔴 Erro ao executar criação de tabela:", execErr)
		return fmt.Errorf("falha ao criar a tabela shares: %w", execErr)
	}

	DB = db
	logs.Info("PostgreSQL conectado e estrutura validada.", nil)
	fmt.Println("✅ Banco conectado e tabela verificada/criada com sucesso.")
	return nil
}

func ClosePostgres() {
	if DB != nil {
		if err := DB.Close(); err != nil {
			logs.Error("Erro ao encerrar conexão com PostgreSQL", map[string]interface{}{"err": err})
		} else {
			logs.Info("Conexão com PostgreSQL encerrada com sucesso.", nil)
		}
	}
}
