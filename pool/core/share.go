package core

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

type ShareStatus int

const (
	ShareAccepted ShareStatus = iota
	ShareDuplicate
	ShareStale
	ShareInvalid
)

type Share struct {
	ID        string    `json:"id,omitempty"`
	JobID     string    `json:"job_id"`
	WorkerID  string    `json:"worker_id"`
	Nonce     string    `json:"nonce"`
	Hash      string    `json:"hash"`
	Diff      float64   `json:"difficulty,omitempty"`
	Timestamp time.Time `json:"timestamp,omitempty"`
	IP        string    `json:"ip,omitempty"`
}

type ShareResult struct {
	Status      ShareStatus `json:"status"`
	Description string      `json:"description"`
	Error       error       `json:"error,omitempty"`
	Valid       bool        `json:"valid"`
}

type ShareValidator interface {
	IsDuplicate(share Share) bool
	IsStale(share Share, jobTTL time.Duration) bool
	IsValidHash(share Share, target string) bool
	ValidateShare(share Share, target string, jobTTL time.Duration) ShareResult
}

type ShareStore interface {
	Exists(shareID string) (bool, error)
	Save(share Share) error
}

type internalStore struct {
	sync.RWMutex
	entries map[string]Share
	ttl     time.Duration
}

func newInternalStore(ttl time.Duration) *internalStore {
	return &internalStore{
		entries: make(map[string]Share),
		ttl:     ttl,
	}
}

func (s *internalStore) Exists(id string) (bool, error) {
	s.cleanup()
	s.RLock()
	defer s.RUnlock()
	_, exists := s.entries[id]
	return exists, nil
}

func (s *internalStore) Save(share Share) error {
	s.Lock()
	defer s.Unlock()
	s.entries[share.ID] = share
	return nil
}

func (s *internalStore) cleanup() {
	s.Lock()
	defer s.Unlock()
	cutoff := time.Now().Add(-s.ttl)
	for id, sh := range s.entries {
		if sh.Timestamp.Before(cutoff) {
			delete(s.entries, id)
		}
	}
}

type DefaultShareValidator struct {
	shareStore ShareStore
}

func NewDefaultShareValidator(store ShareStore) *DefaultShareValidator {
	return &DefaultShareValidator{shareStore: store}
}

func (v *DefaultShareValidator) ValidateShare(share Share, target string, ttl time.Duration) ShareResult {
	switch {
	case v.IsDuplicate(share):
		return ShareResult{Status: ShareDuplicate, Description: "duplicate share", Valid: false}
	case v.IsStale(share, ttl):
		return ShareResult{Status: ShareStale, Description: "stale share", Valid: false}
	case !v.IsValidHash(share, target):
		return ShareResult{Status: ShareInvalid, Description: "invalid hash", Valid: false}
	default:
		return ShareResult{Status: ShareAccepted, Description: "share accepted", Valid: true}
	}
}

func (v *DefaultShareValidator) IsDuplicate(share Share) bool {
	found, _ := v.shareStore.Exists(share.ID)
	return found
}

func (v *DefaultShareValidator) IsStale(share Share, ttl time.Duration) bool {
	return time.Since(share.Timestamp) > ttl
}

func (v *DefaultShareValidator) IsValidHash(share Share, target string) bool {
	expected := calculateHash(share.JobID, share.Nonce)
	return expected < target
}

type ShareProcessor struct {
	validator ShareValidator
	store     ShareStore
}

func NewShareProcessor(v ShareValidator, store ShareStore) *ShareProcessor {
	if store == nil {
		store = newInternalStore(45 * time.Second)
	}
	if v == nil {
		v = NewDefaultShareValidator(store)
	}
	return &ShareProcessor{validator: v, store: store}
}

func (sp *ShareProcessor) Process(share Share, target string, ttl time.Duration) ShareResult {
	result := sp.validator.ValidateShare(share, target, ttl)
	if !result.Valid {
		result.Error = fmt.Errorf("rejected: %s", result.Description)
		return result
	}
	if err := sp.store.Save(share); err != nil {
		return ShareResult{
			Status:      ShareInvalid,
			Description: "save failure",
			Valid:       false,
			Error:       err,
		}
	}
	return result
}

func calculateHash(data, nonce string) string {
	h := sha256.New()
	h.Write([]byte(data + nonce))
	return hex.EncodeToString(h.Sum(nil))
}
