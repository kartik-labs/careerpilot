package browser

import (
	"testing"
	"time"
)

type fakeTokenStore struct {
	hashes map[string]tokenEntry
}

type tokenEntry struct {
	sessionID string
	expiresAt time.Time
	revoked   bool
}

func newFakeTokenStore() *fakeTokenStore {
	return &fakeTokenStore{hashes: map[string]tokenEntry{}}
}

func (s *fakeTokenStore) SaveTokenHash(hash, sessionID string, expiresAt time.Time) error {
	s.hashes[hash] = tokenEntry{sessionID: sessionID, expiresAt: expiresAt}
	return nil
}

func (s *fakeTokenStore) LookupByHash(hash string, now time.Time) (string, bool, error) {
	entry, ok := s.hashes[hash]
	if !ok || entry.revoked || now.After(entry.expiresAt) {
		return "", false, nil
	}
	return entry.sessionID, true, nil
}

func (s *fakeTokenStore) RevokeByHash(hash string) error {
	entry := s.hashes[hash]
	entry.revoked = true
	s.hashes[hash] = entry
	return nil
}

func TestNewRecoveryToken_GeneratesUniqueValues(t *testing.T) {
	now := time.Unix(0, 0)
	t1, err := NewRecoveryToken("session-1", now, time.Hour)
	if err != nil {
		t.Fatalf("NewRecoveryToken() error: %v", err)
	}
	t2, _ := NewRecoveryToken("session-1", now, time.Hour)

	if t1.Raw == t2.Raw {
		t.Error("expected unique raw tokens across calls")
	}
	if t1.Hash != HashRecoveryToken(t1.Raw) {
		t.Error("Hash does not match HashRecoveryToken(Raw)")
	}
	if t1.ExpiresAt != now.Add(time.Hour) {
		t.Errorf("ExpiresAt = %v, want %v", t1.ExpiresAt, now.Add(time.Hour))
	}
}

func TestRedeem_ValidToken(t *testing.T) {
	store := newFakeTokenStore()
	now := time.Unix(0, 0)
	tok, _ := NewRecoveryToken("session-1", now, time.Hour)
	store.SaveTokenHash(tok.Hash, tok.SessionID, tok.ExpiresAt)

	sessionID, err := Redeem(store, tok.Raw, now)
	if err != nil {
		t.Fatalf("Redeem() error: %v", err)
	}
	if sessionID != "session-1" {
		t.Errorf("sessionID = %q, want session-1", sessionID)
	}
}

func TestRedeem_SingleUse(t *testing.T) {
	store := newFakeTokenStore()
	now := time.Unix(0, 0)
	tok, _ := NewRecoveryToken("session-1", now, time.Hour)
	store.SaveTokenHash(tok.Hash, tok.SessionID, tok.ExpiresAt)

	if _, err := Redeem(store, tok.Raw, now); err != nil {
		t.Fatalf("first Redeem() error: %v", err)
	}

	if _, err := Redeem(store, tok.Raw, now); err == nil {
		t.Fatal("second Redeem() expected error (single-use token), got nil")
	}
}

func TestRedeem_ExpiredToken(t *testing.T) {
	store := newFakeTokenStore()
	now := time.Unix(0, 0)
	tok, _ := NewRecoveryToken("session-1", now, time.Minute)
	store.SaveTokenHash(tok.Hash, tok.SessionID, tok.ExpiresAt)

	later := now.Add(2 * time.Minute)
	if _, err := Redeem(store, tok.Raw, later); err == nil {
		t.Fatal("Redeem() expected error for expired token, got nil")
	}
}

func TestRedeem_UnknownToken(t *testing.T) {
	store := newFakeTokenStore()
	if _, err := Redeem(store, "not-a-real-token", time.Unix(0, 0)); err == nil {
		t.Fatal("Redeem() expected error for unknown token, got nil")
	}
}

func TestSession_IsExpired(t *testing.T) {
	now := time.Unix(1000, 0)

	expired := Session{ExpiresAt: time.Unix(500, 0)}
	if !expired.IsExpired(now) {
		t.Error("expected session to be expired")
	}

	notExpired := Session{ExpiresAt: time.Unix(1500, 0)}
	if notExpired.IsExpired(now) {
		t.Error("expected session to not be expired")
	}

	noExpiry := Session{}
	if noExpiry.IsExpired(now) {
		t.Error("zero ExpiresAt should never be considered expired")
	}
}
