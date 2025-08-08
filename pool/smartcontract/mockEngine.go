package smartcontract

import (
	"context"
	"fmt"
	"time"

	"pool/core"
)

type Engine interface {
	Pay(ctx context.Context, share *core.Share) error
	SubmitShare(worker core.WorkerIdentifier, share *core.Share) error
}

type MockEngine struct{}

// Init retorna uma instância do MockEngine
func Init(_ interface{}) (Engine, error) {
	return &MockEngine{}, nil
}

func (m *MockEngine) Pay(ctx context.Context, share *core.Share) error {
	fmt.Printf(
		"[MOCK PAY] [%s] Paying ShareID=%s | WorkerID=%s | Diff=%.5f | Timestamp=%s\n",
		time.Now().Format(time.RFC3339),
		share.ID,
		share.WorkerID,
		share.Diff,
		share.Timestamp.Format(time.RFC3339),
	)
	return nil
}

func (m *MockEngine) SubmitShare(worker core.WorkerIdentifier, share *core.Share) error {
	fmt.Printf(
		"[MOCK SUBMIT] [%s] Worker=%s | ShareID=%s | Nonce=%s | Hash=%s\n",
		time.Now().Format(time.RFC3339),
		worker,
		share.ID,
		share.Nonce,
		share.Hash[:8], // Apenas os primeiros 8 chars do hash
	)
	return nil
}
