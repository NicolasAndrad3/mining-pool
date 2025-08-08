package database

import (
	"context"

	"pool/core"
)

type PostgresShareStore struct {
	db *Postgres
}

func NewPostgresShareStore(pg *Postgres) *PostgresShareStore {
	return &PostgresShareStore{db: pg}
}

func (p *PostgresShareStore) Exists(shareID string) (bool, error) {
	var exists bool
	const query = `SELECT EXISTS(SELECT 1 FROM shares WHERE id = $1)`
	err := p.db.DB.QueryRowContext(context.Background(), query, shareID).Scan(&exists)
	return exists, err
}

func (p *PostgresShareStore) Save(s core.Share) error {
	const query = `
		INSERT INTO shares (id, job_id, worker_id, nonce, hash, difficulty, timestamp, ip)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := p.db.DB.ExecContext(
		context.Background(),
		query,
		s.ID, s.JobID, s.WorkerID, s.Nonce, s.Hash, s.Diff, s.Timestamp, s.IP,
	)
	return err
}
