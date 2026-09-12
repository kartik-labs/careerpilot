package candidate

import "time"

// ClaimSource identifies where a claim's fact originated.
type ClaimSource string

const (
	// SourceProfile means the claim is drawn directly from a Profile field.
	SourceProfile ClaimSource = "profile"
	// SourceUserApproved means the candidate explicitly approved this claim
	// text outside the structured profile (e.g. during a review step).
	SourceUserApproved ClaimSource = "user_approved"
)

// Confidence expresses how certain a claim is, driving whether it can be
// used automatically or must be reviewed. See docs/AGENT-POLICY.md.
type Confidence string

const (
	ConfidenceHigh   Confidence = "high"
	ConfidenceMedium Confidence = "medium"
	ConfidenceLow    Confidence = "low"
)

// Claim is a single factual statement that resume or application content
// may draw from. Every generated claim in a resume must trace back to a
// Claim with AllowedForResume == true; if it cannot, generation must
// return a REVIEW result rather than inventing a replacement.
type Claim struct {
	ID             string
	ProfileID      string
	ProfileVersion int

	Text             string
	Source           ClaimSource
	Confidence       Confidence
	AllowedForResume bool
	Notes            string

	CreatedAt time.Time
}

// ForbiddenClaim is an explicit denylist entry preventing a specific
// fabricated or previously-rejected claim from being reintroduced, even if
// a model proposes it again.
type ForbiddenClaim struct {
	ID        string
	ProfileID string
	Text      string
	Reason    string
	CreatedAt time.Time
}

// Store defines the persistence boundary for candidate profiles and claims.
// Business logic depends on this interface, not on a concrete database
// implementation (CLAUDE.md "interfaces around external systems").
type Store interface {
	GetProfile(profileID string) (Profile, error)
	SaveProfile(p Profile) (Profile, error)

	ListClaims(profileID string) ([]Claim, error)
	SaveClaim(c Claim) (Claim, error)

	ListForbiddenClaims(profileID string) ([]ForbiddenClaim, error)
	AddForbiddenClaim(f ForbiddenClaim) (ForbiddenClaim, error)
}

// IsForbidden reports whether text matches an existing forbidden claim for
// the profile (case-sensitive exact match). Callers needing fuzzier
// duplicate detection should implement it at a higher layer explicitly —
// this stays deterministic and simple by design.
func IsForbidden(text string, forbidden []ForbiddenClaim) bool {
	for _, f := range forbidden {
		if f.Text == text {
			return true
		}
	}
	return false
}
