// Package modelprovider abstracts LLM calls behind a single interface so
// business logic never depends on a vendor SDK directly (CLAUDE.md
// "AI Model Strategy", docs/ARCHITECTURE.md "AI Layer").
package modelprovider

import (
	"context"
	"errors"
	"time"
)

// ErrNotConfigured is returned by placeholder providers that have no real
// backing API call wired up (see gemini.go, claude.go).
var ErrNotConfigured = errors.New("modelprovider: provider not configured")

// Request is a single model call. Prompt is the fully-rendered prompt
// text; callers are responsible for prompt construction so this package
// stays vendor-neutral.
type Request struct {
	Prompt      string
	MaxTokens   int
	Temperature float64

	// Metadata is recorded on the resulting Run for audit purposes
	// (CLAUDE.md "Observability") — e.g. {"purpose": "job_triage", "job_id": "..."}.
	// It must never contain secrets or full candidate PII beyond what the
	// Prompt itself already required.
	Metadata map[string]string
}

// Response is the result of a model call.
type Response struct {
	Text string

	InputTokens  int
	OutputTokens int
}

// Provider is the interface all model vendors implement. Business logic
// depends only on this interface, never on a vendor SDK type
// (docs/ARCHITECTURE.md "AI Layer": "The rest of the system should not
// depend directly on a specific provider.").
type Provider interface {
	// Name identifies the provider for audit/logging (e.g. "gemini", "claude").
	Name() string
	Generate(ctx context.Context, req Request) (Response, error)
}

// Run is an audit record of a single model call, independent of which
// Provider served it. See CLAUDE.md "Observability": model calls must
// record metadata without storing unnecessary sensitive prompts/responses.
type Run struct {
	ID       string
	Provider string
	Purpose  string // e.g. "job_triage", "job_deep_evaluation"

	InputTokens  int
	OutputTokens int
	LatencyMS    int64

	Succeeded     bool
	ErrorCategory string // empty when Succeeded is true

	StartedAt time.Time
}
