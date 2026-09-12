package modelprovider

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeStore struct {
	saved []Run
	err   error
}

func (s *fakeStore) SaveRun(run Run) error {
	if s.err != nil {
		return s.err
	}
	s.saved = append(s.saved, run)
	return nil
}

func idSeq(ids ...string) func() string {
	i := 0
	return func() string {
		id := ids[i]
		i++
		return id
	}
}

func TestRecorder_Generate_Success(t *testing.T) {
	store := &fakeStore{}
	r := NewRecorder(store, idSeq("run-1"))

	provider := &GeminiProvider{GenerateFunc: func(ctx context.Context, req Request) (Response, error) {
		return Response{Text: "ok", InputTokens: 10, OutputTokens: 5}, nil
	}}

	resp, err := r.Generate(context.Background(), provider, "job_triage", Request{Prompt: "hi"})
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}
	if resp.Text != "ok" {
		t.Errorf("Text = %q, want ok", resp.Text)
	}

	if len(store.saved) != 1 {
		t.Fatalf("saved runs = %d, want 1", len(store.saved))
	}
	run := store.saved[0]
	if run.ID != "run-1" || run.Provider != "gemini" || run.Purpose != "job_triage" {
		t.Errorf("run = %+v, unexpected fields", run)
	}
	if !run.Succeeded || run.ErrorCategory != "" {
		t.Errorf("run.Succeeded = %v, ErrorCategory = %q, want true/empty", run.Succeeded, run.ErrorCategory)
	}
	if run.InputTokens != 10 || run.OutputTokens != 5 {
		t.Errorf("tokens = %d/%d, want 10/5", run.InputTokens, run.OutputTokens)
	}
}

func TestRecorder_Generate_ProviderError_StillRecordsRun(t *testing.T) {
	store := &fakeStore{}
	r := NewRecorder(store, idSeq("run-2"))

	provider := &ClaudeProvider{GenerateFunc: func(ctx context.Context, req Request) (Response, error) {
		return Response{}, errors.New("upstream failure")
	}}

	_, err := r.Generate(context.Background(), provider, "job_deep_evaluation", Request{})
	if err == nil {
		t.Fatal("Generate() expected error, got nil")
	}

	if len(store.saved) != 1 {
		t.Fatalf("saved runs = %d, want 1", len(store.saved))
	}
	if store.saved[0].Succeeded {
		t.Error("run.Succeeded = true, want false")
	}
	if store.saved[0].ErrorCategory != "provider_error" {
		t.Errorf("ErrorCategory = %q, want provider_error", store.saved[0].ErrorCategory)
	}
}

func TestRecorder_Generate_TimeoutCategorized(t *testing.T) {
	store := &fakeStore{}
	r := NewRecorder(store, idSeq("run-3"))

	provider := &GeminiProvider{GenerateFunc: func(ctx context.Context, req Request) (Response, error) {
		return Response{}, context.DeadlineExceeded
	}}

	_, _ = r.Generate(context.Background(), provider, "job_triage", Request{})

	if store.saved[0].ErrorCategory != "timeout" {
		t.Errorf("ErrorCategory = %q, want timeout", store.saved[0].ErrorCategory)
	}
}

func TestRecorder_Generate_StoreFailureJoinsError(t *testing.T) {
	store := &fakeStore{err: errors.New("db down")}
	r := NewRecorder(store, idSeq("run-4"))

	provider := &GeminiProvider{GenerateFunc: func(ctx context.Context, req Request) (Response, error) {
		return Response{Text: "ok"}, nil
	}}

	_, err := r.Generate(context.Background(), provider, "job_triage", Request{})
	if err == nil {
		t.Fatal("Generate() expected error when store fails, got nil")
	}
}

func TestUnconfiguredProviders_ReturnErrNotConfigured(t *testing.T) {
	gp := NewGeminiProvider()
	if _, err := gp.Generate(context.Background(), Request{}); !errors.Is(err, ErrNotConfigured) {
		t.Errorf("GeminiProvider error = %v, want ErrNotConfigured", err)
	}

	cp := NewClaudeProvider()
	if _, err := cp.Generate(context.Background(), Request{}); !errors.Is(err, ErrNotConfigured) {
		t.Errorf("ClaudeProvider error = %v, want ErrNotConfigured", err)
	}

	if gp.Name() != "gemini" {
		t.Errorf("Name() = %q, want gemini", gp.Name())
	}
	if cp.Name() != "claude" {
		t.Errorf("Name() = %q, want claude", cp.Name())
	}
}

func TestRecorder_Generate_LatencyRecorded(t *testing.T) {
	store := &fakeStore{}
	r := NewRecorder(store, idSeq("run-5"))
	r.Now = func() time.Time { return time.Unix(0, 0) }

	provider := &GeminiProvider{GenerateFunc: func(ctx context.Context, req Request) (Response, error) {
		return Response{}, nil
	}}

	_, _ = r.Generate(context.Background(), provider, "job_triage", Request{})

	if store.saved[0].LatencyMS != 0 {
		t.Errorf("LatencyMS = %d, want 0 with frozen clock", store.saved[0].LatencyMS)
	}
}
