package lbclient

import (
	"sync"
	"sync/atomic"
	"time"
)

type Selector interface {
	SelectIP() string
	SetFail(ip string)
}

// Roundrobin selector uses next IP in slice (in round) when request comes.
type RoundRobinSelector struct {
	ips []string

	index       uint32
	lock        sync.RWMutex
	failIPs     map[string]time.Time
	failTimeout time.Duration
}

func NewRoundRobinSelector(ips []string) *RoundRobinSelector {
	s := &RoundRobinSelector{
		ips:         ips,
		failIPs:     map[string]time.Time{},
		failTimeout: time.Second,
	}
	return s
}

func (s *RoundRobinSelector) SelectIP() string {
	s.lock.RLock()
	defer s.lock.RUnlock()

	// TODO: panic here if no IP are in the selector?
	if len(s.ips) == 0 {
		return ""
	}

	index := atomic.AddUint32(&s.index, 1)
	candidates := []string{}
	for _, ip := range s.ips {
		_, ok := s.failIPs[ip]
		if !ok {
			candidates = append(candidates, ip)
		}
	}
	// all IPs are marked as fail, still choose one from these IPs.
	if len(candidates) == 0 {
		for _, ip := range s.ips {
			candidates = append(candidates, ip)
		}
	}

	return candidates[index%uint32(len(candidates))]
}

func (s *RoundRobinSelector) SetFail(ip string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.failIPs[ip] = time.Now()
	go func() {
		failTimeout := s.failTimeout
		if failTimeout == 0 {
			failTimeout = time.Second
		}
		<-time.After(failTimeout)
		s.lock.Lock()
		defer s.lock.Unlock()
		now := time.Now()
		t, ok := s.failIPs[ip]
		if ok && now.Sub(t) >= failTimeout {
			delete(s.failIPs, ip)
		}
	}()
}
