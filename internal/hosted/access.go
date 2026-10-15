package hosted

import (
	"crypto/subtle"
	"errors"
	"time"
)

const SessionLifetime = 30 * 24 * time.Hour

type Access struct {
	GrantHash string
	Scope     string
	Sessions  map[string]time.Time
}

// RotateAccess is an operator-only operation: there is no HTTP mint/rotation
// endpoint. The raw capability is returned once and never stored in the store.
func (s *Store) RotateAccess() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data.Schema != 1 || s.failed {
		return "", errors.New("store unavailable")
	}
	secret := randomID()
	next := s.data
	next.Access = Access{GrantHash: hash([]byte(secret)), Scope: randomID(), Sessions: map[string]time.Time{}}
	return secret, s.commit(next)
}
func (s *Store) AccessScope() string { s.mu.Lock(); defer s.mu.Unlock(); return s.data.Access.Scope }
func (s *Store) Exchange(secret string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.failed || s.data.Access.GrantHash == "" {
		return "", fault(401, "access_required")
	}
	digest := hash([]byte(secret))
	if len(secret) != 64 || subtle.ConstantTimeCompare([]byte(digest), []byte(s.data.Access.GrantHash)) != 1 {
		return "", fault(401, "access_required")
	}
	next := s.data
	next.Access.Sessions = map[string]time.Time{}
	for key, expiry := range s.data.Access.Sessions {
		if expiry.After(time.Now()) {
			next.Access.Sessions[key] = expiry
		}
	}
	if len(next.Access.Sessions) >= 100 {
		return "", fault(429, "too_many_browser_sessions")
	}
	token := randomID()
	next.Access.Sessions[hash([]byte(token))] = time.Now().Add(SessionLifetime).UTC()
	if err := s.commit(next); err != nil {
		return "", fault(503, "storage_unavailable")
	}
	return token, nil
}
func (s *Store) Authorized(token string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	expiry, ok := s.data.Access.Sessions[hash([]byte(token))]
	return !s.failed && len(token) == 64 && ok && expiry.After(time.Now())
}
