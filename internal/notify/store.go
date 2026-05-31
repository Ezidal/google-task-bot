package notify

import (
	"sync"
	"time"
)

// Store tracks sent notifications to avoid duplicates across restarts
// only for in-process lifetime (acceptable for personal bot).
type Store struct {
	mu   sync.Mutex
	sent map[string]time.Time
}

func NewStore() *Store {
	return &Store{sent: make(map[string]time.Time)}
}

func (s *Store) Mark(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sent[key] = time.Now()
}

func (s *Store) WasSent(key string, within time.Duration) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.sent[key]
	if !ok {
		return false
	}
	return time.Since(t) < within
}

func (s *Store) ShouldRemindNoDue(key string, interval time.Duration) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.sent[key]
	if !ok {
		return true
	}
	return time.Since(t) >= interval
}
