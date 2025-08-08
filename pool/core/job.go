package core

import (
	"crypto/sha256"
	"encoding/hex"
	"math/rand"
	"sync"
	"time"

	"pool/logs"
	"pool/utils"
)

type Job struct {
	ID          string    `json:"id"`
	Data        string    `json:"data"`
	Target      string    `json:"target"`
	CreatedAt   time.Time `json:"created_at"`
	ExpiresAt   time.Time `json:"expires_at"`
	BlockHeight int       `json:"block_height"`
	Active      bool      `json:"active"`
	Mutex       sync.RWMutex
}

type JobManager struct {
	jobs    map[string]*Job
	lock    sync.RWMutex
	timeout time.Duration
}

func NewJobManager(timeout time.Duration) *JobManager {
	return &JobManager{
		jobs:    make(map[string]*Job),
		timeout: timeout,
	}
}

func (jm *JobManager) CreateJob(blockHeight int) *Job {
	randomSeed := utils.GenerateRandomHex(32)
	data := utils.GenerateRandomHex(64) + randomSeed
	jobID := "job-" + utils.GenerateUUID()
	target := generateTarget(blockHeight)

	job := &Job{
		ID:          jobID,
		Data:        data,
		Target:      target,
		CreatedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(jm.timeout),
		BlockHeight: blockHeight,
		Active:      true,
	}

	jm.lock.Lock()
	defer jm.lock.Unlock()
	jm.jobs[job.ID] = job

	logs.Debugf("New job created: %s | BlockHeight: %d", job.ID, blockHeight)
	return job
}

func (jm *JobManager) ValidateShare(jobID, nonce, result string) bool {
	jm.lock.RLock()
	job, exists := jm.jobs[jobID]
	jm.lock.RUnlock()
	if !exists || !job.Active {
		logs.Warnf("Invalid or inactive job: %s", jobID)
		return false
	}

	job.Mutex.Lock()
	defer job.Mutex.Unlock()
	if time.Now().After(job.ExpiresAt) {
		logs.Infof("Expired job share attempted: %s", jobID)
		return false
	}

	if !utils.ValidateUUID(jobID[4:]) {
		logs.Error("Malformed job ID checksum")
		return false
	}

	expected := hashJob(job.Data, nonce)
	if expected != result {
		logs.Debugf("Invalid share result for job %s | expected: %s, got: %s", jobID, expected, result)
		return false
	}

	if !checkDifficulty(expected, job.Target) {
		logs.Debugf("Share did not meet target for job %s", jobID)
		return false
	}

	return true
}

func (jm *JobManager) CleanupExpiredJobs() {
	jm.lock.Lock()
	defer jm.lock.Unlock()
	now := time.Now()
	for id, job := range jm.jobs {
		if now.After(job.ExpiresAt) {
			job.Active = false
			delete(jm.jobs, id)
			logs.Debugf("Expired job removed: %s", id)
		}
	}
}

func (jm *JobManager) StartGarbageCollector(interval time.Duration) {
	go func() {
		for {
			time.Sleep(interval)
			jm.CleanupExpiredJobs()
		}
	}()
}

func hashJob(data, nonce string) string {
	h := sha256.New()
	h.Write([]byte(data + nonce))
	return hex.EncodeToString(h.Sum(nil))
}

func checkDifficulty(hash, target string) bool {
	return hash < target
}

func generateTarget(blockHeight int) string {
	base := "0000ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
	if blockHeight%10 == 0 {
		return "00000ffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
	}
	if rand.Float64() < 0.05 {
		return "000000ffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
	}
	return base
}
