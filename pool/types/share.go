package types

import "time"

type Share struct {
	ID        string
	WorkerID  string
	JobID     string
	Nonce     string
	Hash      string
	Diff      float64
	Timestamp time.Time
	IP        string
}
