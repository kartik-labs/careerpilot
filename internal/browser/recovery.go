package browser

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"
)

// RecoveryToken is an opaque, single-purpose credential handed to the
// user so they can complete a paused browser step (e.g. a CAPTCHA) and
// have the session resume. The raw token is shown to the user exactly
// once; only its hash is ever persisted (CLAUDE.md/docs/BROWSER-RECOVERY.md
// "Recovery Link": "cryptographically random", "hashed at rest",
// "short-lived", "revocable").
type RecoveryToken struct {
	Raw       string // returned to the caller once; never stored
	Hash      string // what gets persisted
	SessionID string
	ExpiresAt time.Time
}

const recoveryTokenBytes = 32

// NewRecoveryToken generates a fresh random token for sessionID, valid
// until now+ttl.
func NewRecoveryToken(sessionID string, now time.Time, ttl time.Duration) (RecoveryToken, error) {
	buf := make([]byte, recoveryTokenBytes)
	if _, err := rand.Read(buf); err != nil {
		return RecoveryToken{}, fmt.Errorf("browser: generate recovery token: %w", err)
	}
	raw := base64.RawURLEncoding.EncodeToString(buf)

	return RecoveryToken{
		Raw:       raw,
		Hash:      HashRecoveryToken(raw),
		SessionID: sessionID,
		ExpiresAt: now.Add(ttl),
	}, nil
}

// HashRecoveryToken deterministically hashes a raw token for storage and
// lookup. Recovery tokens are high-entropy random values, so a fast hash
// (SHA-256) is appropriate here — unlike passwords, there is no
// brute-force risk from low entropy.
func HashRecoveryToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// RecoveryTokenStore persists only token hashes, never raw tokens.
type RecoveryTokenStore interface {
	SaveTokenHash(hash, sessionID string, expiresAt time.Time) error
	// LookupByHash returns the sessionID for a hash, and whether it was
	// found and not expired/revoked.
	LookupByHash(hash string, now time.Time) (sessionID string, ok bool, err error)
	RevokeByHash(hash string) error
}

// Redeem validates a raw recovery token against the store and, if valid,
// revokes it (single-use) and returns the associated sessionID.
func Redeem(store RecoveryTokenStore, rawToken string, now time.Time) (string, error) {
	hash := HashRecoveryToken(rawToken)

	sessionID, ok, err := store.LookupByHash(hash, now)
	if err != nil {
		return "", fmt.Errorf("browser: redeem recovery token: %w", err)
	}
	if !ok {
		return "", fmt.Errorf("browser: recovery token invalid or expired")
	}

	if err := store.RevokeByHash(hash); err != nil {
		return "", fmt.Errorf("browser: revoke recovery token after redemption: %w", err)
	}

	return sessionID, nil
}
