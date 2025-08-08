package core

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"pool/config"
	"pool/metrics"
	"pool/security"
)

type (
	WorkerIdentifier string
	JobIdentifier    string
)

type DB interface{}

type PaymentEngine interface {
	Pay(ctx context.Context, share *Share) error
}

type PoolCore struct {
	workers        map[WorkerIdentifier]*Worker
	activeJobs     map[JobIdentifier]*Job
	workerShares   map[WorkerIdentifier][]*Share
	lastSubmission map[WorkerIdentifier]time.Time

	ttlJob   time.Duration
	interval time.Duration

	muShares sync.Mutex
	muJobs   sync.RWMutex
	muWrks   sync.RWMutex
}

func InitPool(ttl, rate time.Duration) *PoolCore {
	return &PoolCore{
		workers:        make(map[WorkerIdentifier]*Worker),
		activeJobs:     make(map[JobIdentifier]*Job),
		workerShares:   make(map[WorkerIdentifier][]*Share),
		lastSubmission: make(map[WorkerIdentifier]time.Time),
		ttlJob:         ttl,
		interval:       rate,
	}
}

func (pc *PoolCore) AppendWorker(id WorkerIdentifier, wrk *Worker) error {
	if id == "" || wrk == nil {
		return errors.New("worker invalid")
	}
	pc.muWrks.Lock()
	defer pc.muWrks.Unlock()
	pc.workers[id] = wrk
	return nil
}

func (pc *PoolCore) FetchWorker(id WorkerIdentifier) (*Worker, bool) {
	pc.muWrks.RLock()
	defer pc.muWrks.RUnlock()
	w, ok := pc.workers[id]
	return w, ok
}

func (pc *PoolCore) PurgeWorker(id WorkerIdentifier) {
	pc.muWrks.Lock()
	delete(pc.workers, id)
	pc.muWrks.Unlock()

	pc.muShares.Lock()
	delete(pc.workerShares, id)
	delete(pc.lastSubmission, id)
	pc.muShares.Unlock()
}

func (pc *PoolCore) DispatchJob(j *Job) {
	if j == nil || j.ID == "" {
		return
	}
	pc.muJobs.Lock()
	pc.activeJobs[JobIdentifier(j.ID)] = j
	pc.muJobs.Unlock()
	pc.removeExpiredJobs()
}

func (pc *PoolCore) SubmitShare(wid WorkerIdentifier, s *Share) error {
	if s == nil {
		return errors.New("nil share")
	}

	pc.muJobs.RLock()
	job, exists := pc.activeJobs[JobIdentifier(s.JobID)]
	pc.muJobs.RUnlock()
	if !exists || time.Since(job.CreatedAt) > pc.ttlJob {
		return errors.New("invalid or expired job")
	}

	pc.muWrks.RLock()
	worker, ok := pc.workers[wid]
	pc.muWrks.RUnlock()
	if !ok || !worker.IsActive() {
		return errors.New("worker not active")
	}

	now := time.Now()
	pc.muShares.Lock()
	defer pc.muShares.Unlock()

	if last, ok := pc.lastSubmission[wid]; ok {
		if now.Sub(last) < pc.interval {
			return errors.New("rate limit exceeded")
		}
	}
	pc.lastSubmission[wid] = now
	pc.workerShares[wid] = append(pc.workerShares[wid], s)
	return nil
}

func (pc *PoolCore) ListShares(wid WorkerIdentifier) ([]*Share, bool) {
	pc.muShares.Lock()
	defer pc.muShares.Unlock()
	list, ok := pc.workerShares[wid]
	return list, ok
}

func (pc *PoolCore) ActiveJobList() []*Job {
	pc.muJobs.RLock()
	defer pc.muJobs.RUnlock()
	out := make([]*Job, 0, len(pc.activeJobs))
	for _, job := range pc.activeJobs {
		out = append(out, job)
	}
	return out
}

func (pc *PoolCore) removeExpiredJobs() {
	threshold := time.Now().Add(-pc.ttlJob)
	pc.muJobs.Lock()
	defer pc.muJobs.Unlock()
	for id, job := range pc.activeJobs {
		if job.CreatedAt.Before(threshold) {
			delete(pc.activeJobs, id)
		}
	}
}

type Pool struct {
	cfg        *config.Config
	db         DB
	payment    PaymentEngine
	Engine     *PoolCore
	ShareStore ShareStore
}

func NewPool(cfg *config.Config, db DB, sc PaymentEngine, store ShareStore) *Pool {
	return &Pool{
		cfg:        cfg,
		db:         db,
		payment:    sc,
		Engine:     InitPool(30*time.Second, 2*time.Second),
		ShareStore: store,
	}
}

func (p *Pool) ProcessShare(ctx context.Context, s Share) (ShareResult, error) {
	start := time.Now()
	defer func() {
		if metrics.ValidationDuration != nil {
			metrics.ValidationDuration.Observe(time.Since(start).Seconds())
		}
	}()

	if err := validateShareBasic(&s); err != nil {
		if metrics.SharesInvalid != nil {
			metrics.SharesInvalid.Inc()
		}
		return ShareResult{
			Status:      ShareInvalid,
			Description: "basic validation failed",
			Error:       err,
			Valid:       false,
		}, err
	}

	if !p.isJobActive(s.JobID) {
		if metrics.SharesInvalid != nil {
			metrics.SharesInvalid.Inc()
		}
		return ShareResult{
			Status:      ShareInvalid, // poderia ser ShareStale dependendo da semântica
			Description: "job not active or expired",
			Error:       errors.New("job not active or expired"),
			Valid:       false,
		}, nil
	}

	if verdict := security.EvaluateShare(s.WorkerID, s.IP, s.Nonce, s.Hash, s.Timestamp); verdict.Flagged {
		if metrics.SharesInvalid != nil {
			metrics.SharesInvalid.Inc()
		}
		return ShareResult{
			Status:      ShareInvalid,
			Description: fmt.Sprintf("blocked by antifraud: %s", verdict.Reason),
			Error:       errors.New("antifraud rejection"),
			Valid:       false,
		}, nil
	}

	if p.ShareStore != nil {
		if exists, err := p.ShareStore.Exists(s.ID); err == nil && exists {
			return ShareResult{
				Status:      ShareAccepted, // mantém aceito/ignorado para UX
				Description: "duplicate share ignored",
				Valid:       true,
			}, nil
		}
	}

	if p.ShareStore != nil {
		if err := p.ShareStore.Save(s); err != nil {
			if metrics.SharesInvalid != nil {
				metrics.SharesInvalid.Inc()
			}
			return ShareResult{
				Status:      ShareInvalid,
				Description: "failed to persist share",
				Error:       err,
				Valid:       false,
			}, err
		}
	}

	if metrics.SharesValid != nil {
		metrics.SharesValid.Inc()
	}

	return ShareResult{
		Status:      ShareAccepted,
		Description: "share accepted",
		Error:       nil,
		Valid:       true,
	}, nil
}

func (p *Pool) isJobActive(jobID string) bool {
	if jobID == "" {
		return false
	}
	p.Engine.muJobs.RLock()
	defer p.Engine.muJobs.RUnlock()
	j, ok := p.Engine.activeJobs[JobIdentifier(jobID)]
	if !ok {
		return false
	}
	return time.Since(j.CreatedAt) <= p.Engine.ttlJob
}

func validateShareBasic(s *Share) error {
	if s.JobID == "" || s.WorkerID == "" || s.Nonce == "" || s.Hash == "" {
		return errors.New("missing required fields")
	}
	if s.Timestamp.IsZero() || time.Since(s.Timestamp) > 5*time.Minute {
		return errors.New("timestamp out of range")
	}
	return nil
}

func (p *Pool) Start(ctx context.Context) {
	t := time.NewTicker(15 * time.Second)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			p.Engine.removeExpiredJobs()
		}
	}
}
