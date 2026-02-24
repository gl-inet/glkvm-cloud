package memory

import (
	"sync"
	"time"
)

type Session struct {
	Token     string
	UserID    int64
	ExpiresAt time.Time
}

type SessionStore struct {
	mu   sync.RWMutex
	ttl  time.Duration
	data map[string]Session
}

func NewSessionStore(ttl time.Duration) *SessionStore {
	s := &SessionStore{
		ttl:  ttl,
		data: map[string]Session{},
	}
	go s.gcLoop()
	return s
}

func (s *SessionStore) Create(token string, userID int64) Session {
	sess := Session{Token: token, UserID: userID, ExpiresAt: time.Now().Add(s.ttl)}
	s.mu.Lock()
	s.data[token] = sess
	s.mu.Unlock()
	return sess
}

func (s *SessionStore) Get(token string) (Session, bool) {
	s.mu.RLock()
	sess, ok := s.data[token]
	s.mu.RUnlock()
	if !ok {
		return Session{}, false
	}
	if time.Now().After(sess.ExpiresAt) {
		s.Delete(token)
		return Session{}, false
	}
	return sess, true
}

func (s *SessionStore) Delete(token string) {
	s.mu.Lock()
	delete(s.data, token)
	s.mu.Unlock()
}

func (s *SessionStore) DeleteByUserID(userID int64) {
	s.mu.Lock()
	for k, v := range s.data {
		if v.UserID == userID {
			delete(s.data, k)
		}
	}
	s.mu.Unlock()
}

func (s *SessionStore) gcLoop() {
	t := time.NewTicker(2 * time.Minute)
	defer t.Stop()
	for range t.C {
		now := time.Now()
		s.mu.Lock()
		for k, v := range s.data {
			if now.After(v.ExpiresAt) {
				delete(s.data, k)
			}
		}
		s.mu.Unlock()
	}
}
