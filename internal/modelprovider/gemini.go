package modelprovider

import "context"

// GeminiProvider is a placeholder implementation of Provider for Google
// Gemini. No Gemini SDK/API credentials are available in this
// environment, so this is a deterministic mock rather than a real HTTP
// integration — replace GenerateFunc (or add a real HTTP-backed
// implementation behind this same interface) once API access exists.
//
// CLAUDE.md routes Gemini to high-volume/inexpensive work (job triage);
// see docs/IMPLEMENTATION-PLAN.md section 2.2.
type GeminiProvider struct {
	// GenerateFunc, if set, is called by Generate. Tests and callers
	// without real API access can supply canned responses here.
	GenerateFunc func(ctx context.Context, req Request) (Response, error)
}

// NewGeminiProvider returns a GeminiProvider. Without a GenerateFunc it
// returns ErrNotConfigured on every call, so misuse fails loudly instead
// of silently returning fabricated data.
func NewGeminiProvider() *GeminiProvider {
	return &GeminiProvider{}
}

func (p *GeminiProvider) Name() string { return "gemini" }

func (p *GeminiProvider) Generate(ctx context.Context, req Request) (Response, error) {
	if p.GenerateFunc == nil {
		return Response{}, ErrNotConfigured
	}
	return p.GenerateFunc(ctx, req)
}
