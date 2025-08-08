package security

import (
	"net"
	"strings"
	"sync"
	"time"
)

const (
	_windowSpan           = 15 * time.Second
	_maxRequestsPerSubnet = 20
	_nonceReuseWarn       = 3
	_nonceReuseBlock      = 10
	_greylistTimeout      = 3 * time.Minute
	_clockSkewTolerance   = 2 * time.Minute
)

type ThreatLevel int

const (
	NoThreat ThreatLevel = iota
	Warn
	Block
)

type Verdict struct {
	Flagged bool
	Reason  string
	Level   ThreatLevel
}

type record struct {
	Timestamps []time.Time
}

type usage struct {
	Entries []time.Time
}

type state struct {
	Greylist map[string]time.Time
	Subnets  map[string]record
	Tokens   map[string]usage
	NonceMap map[string]map[string]time.Time
}

type Inspector struct {
	lock    sync.Mutex
	current state
}

var (
	globalInspector *Inspector
	once            sync.Once
)

func LaunchInspector() *Inspector {
	once.Do(func() {
		globalInspector = &Inspector{
			current: state{
				Greylist: make(map[string]time.Time),
				Subnets:  make(map[string]record),
				Tokens:   make(map[string]usage),
				NonceMap: make(map[string]map[string]time.Time),
			},
		}
		go globalInspector.cleanGreylist()
	})
	return globalInspector
}

func EvaluateShare(minerID, ip, nonce, hash string, ts time.Time) Verdict {
	if globalInspector == nil {
		LaunchInspector()
	}

	now := time.Now()
	if ts.IsZero() || ts.After(now.Add(_clockSkewTolerance)) || now.Sub(ts) > _clockSkewTolerance {
		return Verdict{Flagged: true, Reason: "clock skew out of tolerance", Level: Warn}
	}

	subnet := extractSubnet(ip)
	lvl := globalInspector.LogRequest(subnet, nonce)
	if lvl == Block {
		return Verdict{Flagged: true, Reason: "rate limit subnet/24 exceeded (greylisted)", Level: Block}
	}
	if lvl == Warn {
	}

	nlvl, nreason := globalInspector.checkNonceReuse(minerID, nonce, now)
	switch nlvl {
	case Block:
		return Verdict{Flagged: true, Reason: nreason, Level: Block}
	case Warn:
		return Verdict{Flagged: true, Reason: nreason, Level: Warn}
	default:
	}

	hlvl, hreason := globalInspector.markToken(hash, now)
	if hlvl == Block {
		return Verdict{Flagged: true, Reason: hreason, Level: Block}
	}
	if hlvl == Warn {
		return Verdict{Flagged: true, Reason: hreason, Level: Warn}
	}

	if globalInspector.Check(subnet) {
		return Verdict{Flagged: true, Reason: "subnet greylisted", Level: Block}
	}

	return Verdict{Flagged: false, Reason: "", Level: NoThreat}
}

func IsFraudulentNonce(minerID, nonce string) bool {
	if globalInspector == nil {
		LaunchInspector()
	}
	verdict := EvaluateShare(minerID, minerID, nonce, "", time.Now())
	return verdict.Level == Block
}

func (i *Inspector) cleanGreylist() {
	tick := time.NewTicker(90 * time.Second)
	defer tick.Stop()
	for range tick.C {
		now := time.Now()
		i.lock.Lock()
		for subnet, added := range i.current.Greylist {
			if now.Sub(added) > _greylistTimeout {
				delete(i.current.Greylist, subnet)
			}
		}
		i.lock.Unlock()
	}
}

func (i *Inspector) Check(subnet string) bool {
	i.lock.Lock()
	defer i.lock.Unlock()

	added, exists := i.current.Greylist[subnet]
	if !exists {
		return false
	}
	return time.Since(added) <= _greylistTimeout
}

func (i *Inspector) LogRequest(ipOrSubnet string, token string) ThreatLevel {
	subnet := extractSubnet(ipOrSubnet)
	now := time.Now()

	i.lock.Lock()
	defer i.lock.Unlock()

	rec := i.current.Subnets[subnet]
	rec.Timestamps = pruneOld(rec.Timestamps, now)
	rec.Timestamps = append(rec.Timestamps, now)
	i.current.Subnets[subnet] = rec

	u := i.current.Tokens[token]
	u.Entries = pruneOld(u.Entries, now)
	u.Entries = append(u.Entries, now)
	i.current.Tokens[token] = u

	if len(rec.Timestamps) > _maxRequestsPerSubnet {
		i.current.Greylist[subnet] = now
		return Block
	}
	if len(u.Entries) >= _nonceReuseWarn && len(u.Entries) < _nonceReuseBlock {
		return Warn
	}
	if len(u.Entries) >= _nonceReuseBlock {
		return Block
	}
	return NoThreat
}

func (i *Inspector) checkNonceReuse(minerID, nonce string, now time.Time) (ThreatLevel, string) {
	i.lock.Lock()
	defer i.lock.Unlock()

	if _, ok := i.current.NonceMap[minerID]; !ok {
		i.current.NonceMap[minerID] = make(map[string]time.Time)
	}

	for n, t := range i.current.NonceMap[minerID] {
		if now.Sub(t) > _windowSpan {
			delete(i.current.NonceMap[minerID], n)
		}
	}

	_, seen := i.current.NonceMap[minerID][nonce]
	i.current.NonceMap[minerID][nonce] = now

	if !seen {
		return NoThreat, ""
	}

	return Warn, "nonce reuse by miner within window"
}

func (i *Inspector) markToken(token string, now time.Time) (ThreatLevel, string) {
	if token == "" {
		return NoThreat, ""
	}

	i.lock.Lock()
	defer i.lock.Unlock()

	u := i.current.Tokens[token]
	u.Entries = pruneOld(u.Entries, now)
	u.Entries = append(u.Entries, now)
	i.current.Tokens[token] = u

	if len(u.Entries) >= _nonceReuseBlock {
		return Block, "hash reuse over hard threshold"
	}
	if len(u.Entries) >= _nonceReuseWarn {
		return Warn, "hash reuse approaching threshold"
	}
	return NoThreat, ""
}

func extractSubnet(ipStr string) string {
	ip := net.ParseIP(strings.TrimSpace(ipStr))
	if ip == nil {
		return ipStr
	}
	if ipv4 := ip.To4(); ipv4 != nil {
		mask := net.CIDRMask(24, 32)
		return ipv4.Mask(mask).String()
	}
	return ip.String()
}

func pruneOld(ts []time.Time, now time.Time) []time.Time {
	filtered := ts[:0]
	for _, t := range ts {
		if now.Sub(t) <= _windowSpan {
			filtered = append(filtered, t)
		}
	}
	return filtered
}
