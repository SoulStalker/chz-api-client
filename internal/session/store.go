package session

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

const ttl = 10 * time.Hour

type Store struct {
	mu      sync.RWMutex
	entries map[string]entry
}

type entry struct {
	token     string
	expiresAt time.Time
}

func NewStore() *Store {
	s := &Store{entries: make(map[string]entry)}
	go s.cleanup()
	return s
}

// Set сохраняет токен и возвращает ID сессии.
func (s *Store) Set(token string) string {
	id := uuid.New().String()
	s.mu.Lock()
	s.entries[id] = entry{token: token, expiresAt: time.Now().Add(ttl)}
	s.mu.Unlock()
	return id
}

// Get возвращает токен по ID сессии.
func (s *Store) Get(id string) (string, bool) {
	s.mu.RLock()
	e, ok := s.entries[id]
	s.mu.RUnlock()
	if !ok || time.Now().After(e.expiresAt) {
		return "", false
	}
	return e.token, true
}

// Delete удаляет сессию.
func (s *Store) Delete(id string) {
	s.mu.Lock()
	delete(s.entries, id)
	s.mu.Unlock()
}

func (s *Store) cleanup() {
	for range time.Tick(time.Hour) {
		s.mu.Lock()
		for id, e := range s.entries {
			if time.Now().After(e.expiresAt) {
				delete(s.entries, id)
			}
		}
		s.mu.Unlock()
	}
}
