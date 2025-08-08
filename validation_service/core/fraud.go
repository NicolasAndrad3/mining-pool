package core

import (
	"net"
	"strings"
	"sync"
	"time"
)

type ShareSubmission struct {
	WorkerID    string
	ShareHash   string
	SubmittedAt time.Time
	IPAddress   string
}

type RiskReport struct {
	IsFraudulent bool
	Score        float64
	Reason       string
}

type ShareFraudDetector struct {
	history   map[string][]ShareSubmission
	hashCache map[string]time.Time
	lock      sync.RWMutex
}

func InitFraudModule() *ShareFraudDetector {
	return &ShareFraudDetector{
		history:   make(map[string][]ShareSubmission),
		hashCache: make(map[string]time.Time),
	}
}

func (fd *ShareFraudDetector) Analyze(sub ShareSubmission) RiskReport {
	fd.lock.Lock()
	defer fd.lock.Unlock()

	var (
		infractions float64
		violations  []string
		now         = time.Now()
	)

	// Constants for scoring
	const (
		riskDuplicateHash = 2.5
		riskHighFrequency = 1.5
		riskIPMismatch    = 1.0
	)

	if t, exists := fd.hashCache[sub.ShareHash]; exists && now.Sub(t) < 30*time.Second {
		infractions += riskDuplicateHash
		violations = append(violations, "Repeated share hash")
	} else {
		fd.hashCache[sub.ShareHash] = now
	}

	recent := fd.history[sub.WorkerID]
	fd.history[sub.WorkerID] = append(recent, sub)

	if len(recent) >= 3 {
		interval := now.Sub(recent[len(recent)-3].SubmittedAt)
		if interval < 2*time.Second {
			infractions += riskHighFrequency
			violations = append(violations, "Rapid share frequency")
		}
	}

	for _, r := range recent {
		if !inSameSubnet(r.IPAddress, sub.IPAddress) {
			infractions += riskIPMismatch
			violations = append(violations, "IP/subnet inconsistency")
			break
		}
	}

	for hash, t := range fd.hashCache {
		if now.Sub(t) > 60*time.Second {
			delete(fd.hashCache, hash)
		}
	}

	return RiskReport{
		IsFraudulent: infractions >= 3.0,
		Score:        infractions,
		Reason:       strings.Join(violations, "; "),
	}
}

func inSameSubnet(a, b string) bool {
	ip1, ip2 := net.ParseIP(a).To4(), net.ParseIP(b).To4()
	if ip1 == nil || ip2 == nil {
		return false
	}
	for i := 0; i < 3; i++ {
		if ip1[i] != ip2[i] {
			return false
		}
	}
	return true
}
