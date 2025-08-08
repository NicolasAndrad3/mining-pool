package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"time"

	"pool/core"
	"pool/logs"

	_ "github.com/lib/pq"
)

type ctxKey string

const RequestIDKey ctxKey = "request_id"

type Postgres struct {
	DB *sql.DB
}

func Initialize(dsn string) (*Postgres, error) {
	if dsn == "" {
		return nil, errors.New("missing database DSN")
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("unable to open db connection: %w", err)
	}

	db.SetMaxOpenConns(50)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(45 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping error: %w", err)
	}

	logs.WithFields(map[string]interface{}{
		"component": "postgres",
	}).Info("Database connection established")

	return &Postgres{DB: db}, nil
}

func Connect(dsn string) (*Postgres, error) {
	return Initialize(dsn)
}

func ConnectFromEnv() *Postgres {
	envs := []string{"DATABASE_URL", "POSTGRES_DSN", "DB_CONN"}
	var dsn string
	for _, env := range envs {
		if v := os.Getenv(env); v != "" {
			dsn = v
			break
		}
	}
	if dsn == "" {
		logs.WithFields(map[string]interface{}{
			"component": "postgres",
		}).Fatal("No valid environment variable found for DB connection")
	}
	pg, err := Initialize(dsn)
	if err != nil {
		logs.WithFields(map[string]interface{}{
			"component": "postgres",
			"error":     err.Error(),
		}).Fatal("Failed to connect to database")
	}
	return pg
}

func (p *Postgres) HealthCheck(ctx context.Context) error {
	hCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return p.DB.PingContext(hCtx)
}

func (p *Postgres) Close() error {
	if err := p.DB.Close(); err != nil {
		logs.WithFields(map[string]interface{}{
			"component": "postgres",
			"error":     err.Error(),
		}).Error("Error closing DB connection")
		return fmt.Errorf("close error: %w", err)
	}
	logs.WithFields(map[string]interface{}{
		"component": "postgres",
	}).Warn("Database connection closed")
	return nil
}

func (p *Postgres) Exists(shareID string) (bool, error) {
	var exists bool
	err := p.DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM shares WHERE id=$1)`, shareID).Scan(&exists)
	if err != nil {
		logs.WithFields(map[string]interface{}{
			"component": "postgres",
			"share_id":  shareID,
			"error":     err.Error(),
		}).Error("Failed to check share existence")
		return false, err
	}
	return exists, nil
}

func (p *Postgres) Save(s core.Share) error {
	_, err := p.DB.Exec(`
		INSERT INTO shares (id, job_id, worker_id, nonce, hash, difficulty, timestamp, ip)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, s.ID, s.JobID, s.WorkerID, s.Nonce, s.Hash, s.Diff, s.Timestamp, s.IP)

	if err != nil {
		logs.WithFields(map[string]interface{}{
			"component": "postgres",
			"share_id":  s.ID,
			"error":     err.Error(),
		}).Error("Failed to persist share")
	}
	return err
}

func (p *Postgres) Exec(ctx context.Context, query string, args ...any) error {
	qCtx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()

	reqID := ctx.Value(RequestIDKey)

	stmt, err := p.DB.PrepareContext(qCtx, query)
	if err != nil {
		logs.WithFields(map[string]interface{}{
			"request_id": reqID,
			"query":      query,
			"error":      err.Error(),
		}).Error("Statement preparation failed")
		return fmt.Errorf("prepare failed: %w", err)
	}
	defer stmt.Close()

	if _, err := stmt.ExecContext(qCtx, args...); err != nil {
		logs.WithFields(map[string]interface{}{
			"request_id": reqID,
			"query":      query,
			"error":      err.Error(),
		}).Error("Statement execution failed")
		return fmt.Errorf("exec failed: %w", err)
	}

	return nil
}

func (p *Postgres) QueryRow(ctx context.Context, query string, args ...any) *sql.Row {
	return p.DB.QueryRowContext(ctx, query, args...)
}
