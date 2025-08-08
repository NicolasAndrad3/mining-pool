package core

import (
	"math/rand"
	"strconv"
	"sync"
	"time"
)

type PoolStats struct {
	TotalWorkers  int     `json:"total_workers"`
	TotalHashrate float64 `json:"total_hashrate"`
	Uptime        string  `json:"uptime"`
}

var (
	startTime   time.Time
	statsLock   sync.RWMutex
	defaultPool *PoolCore
)

func init() {
	startTime = time.Now()
	rand.Seed(time.Now().UnixNano())
	defaultPool = InitPool(60*time.Second, 5*time.Second)
}

func GetCurrentPoolStats() PoolStats {
	statsLock.RLock()
	defer statsLock.RUnlock()

	workers := fetchWorkerCount()
	hashrate := estimateHashrate(workers)
	uptime := formatDuration(time.Since(startTime))

	return PoolStats{
		TotalWorkers:  workers,
		TotalHashrate: hashrate,
		Uptime:        uptime,
	}
}

func fetchWorkerCount() int {
	if defaultPool == nil {
		return 0
	}

	defaultPool.muWrks.RLock()
	defer defaultPool.muWrks.RUnlock()

	return len(defaultPool.workers)
}

func estimateHashrate(workers int) float64 {
	if workers == 0 {
		return 0
	}
	base := 350.0 + rand.Float64()*120.0
	return float64(workers) * base
}

func formatDuration(d time.Duration) string {
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	mins := int(d.Minutes()) % 60

	var parts []string
	if days > 0 {
		parts = append(parts, strconv.Itoa(days)+"d")
	}
	if hours > 0 {
		parts = append(parts, strconv.Itoa(hours)+"h")
	}
	if mins > 0 {
		parts = append(parts, strconv.Itoa(mins)+"m")
	}
	if len(parts) == 0 {
		return "0m"
	}
	return customJoin(parts)
}

func customJoin(elements []string) string {
	out := ""
	for _, el := range elements {
		out += el
	}
	return out
}
