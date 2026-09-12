package modelprovider

import (
	"context"
	"errors"
	"time"
)

// RunStore persists model_runs audit records (docs/IMPLEMENTATION-PLAN.md
// "Storage" / CLAUDE.md "Observability").
type RunStore interface {
	SaveRun(run Run) error
}

// Recorder wraps a Provider so every call is audited via RunStore,
// regardless of success or failure. This is the only place Run records are
// produced — deterministic Go code, never the model itself, decides what
// gets audited (CLAUDE.md "Observability").
type Recorder struct {
	Store RunStore
	NewID func() string
	Now   func() time.Time
}

// NewRecorder builds a Recorder. newID must return a unique ID per call
// (e.g. a UUID generator); it is injected so tests are deterministic.
func NewRecorder(store RunStore, newID func() string) *Recorder {
	return &Recorder{
		Store: store,
		NewID: newID,
		Now:   time.Now,
	}
}

// Generate calls provider.Generate, records a Run, and returns the
// provider's result unchanged.
func (r *Recorder) Generate(ctx context.Context, provider Provider, purpose string, req Request) (Response, error) {
	started := r.Now()

	resp, err := provider.Generate(ctx, req)

	run := Run{
		ID:           r.NewID(),
		Provider:     provider.Name(),
		Purpose:      purpose,
		InputTokens:  resp.InputTokens,
		OutputTokens: resp.OutputTokens,
		LatencyMS:    r.Now().Sub(started).Milliseconds(),
		Succeeded:    err == nil,
		StartedAt:    started,
	}
	if err != nil {
		run.ErrorCategory = categorizeError(err)
	}

	if saveErr := r.Store.SaveRun(run); saveErr != nil {
		return resp, errors.Join(err, saveErr)
	}

	return resp, err
}

func categorizeError(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}
	if errors.Is(err, context.Canceled) {
		return "canceled"
	}
	return "provider_error"
}
