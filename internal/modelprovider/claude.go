package modelprovider

import "context"

// ClaudeProvider is a placeholder implementation of Provider for
// Anthropic Claude. No Claude API credentials are available in this
// environment, so this is a deterministic mock rather than a real HTTP
// integration — replace GenerateFunc (or add a real HTTP-backed
// implementation behind this same interface) once API access exists.
//
// CLAUDE.md routes Claude to deep evaluation, resume tailoring, and
// complex reasoning; see docs/IMPLEMENTATION-PLAN.md section 2.3.
type ClaudeProvider struct {
	GenerateFunc func(ctx context.Context, req Request) (Response, error)
}

// NewClaudeProvider returns a ClaudeProvider. Without a GenerateFunc it
// returns ErrNotConfigured on every call.
func NewClaudeProvider() *ClaudeProvider {
	return &ClaudeProvider{}
}

func (p *ClaudeProvider) Name() string { return "claude" }

func (p *ClaudeProvider) Generate(ctx context.Context, req Request) (Response, error) {
	if p.GenerateFunc == nil {
		return Response{}, ErrNotConfigured
	}
	return p.GenerateFunc(ctx, req)
}
