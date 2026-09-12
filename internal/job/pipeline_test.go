package job

import (
	"context"
	"testing"
	"time"

	"github.com/kartik-labs/careerpilot/internal/modelprovider"
)

type fakeEvalStore struct {
	evaluations map[string]Evaluation // key: jobID+"|"+stage
	deepCount   int
}

func newFakeEvalStore() *fakeEvalStore {
	return &fakeEvalStore{evaluations: map[string]Evaluation{}}
}

func (s *fakeEvalStore) SaveEvaluation(e Evaluation) error {
	s.evaluations[e.JobID+"|"+e.Stage] = e
	if e.Stage == "deep_evaluation" {
		s.deepCount++
	}
	return nil
}

func (s *fakeEvalStore) CountDeepEvaluationsSince(since time.Time) (int, error) {
	return s.deepCount, nil
}

func (s *fakeEvalStore) GetEvaluation(jobID, stage string) (Evaluation, bool, error) {
	e, ok := s.evaluations[jobID+"|"+stage]
	return e, ok, nil
}

type fakePromptBuilder struct{}

func (fakePromptBuilder) TriagePrompt(j Job) string     { return "triage:" + j.ID }
func (fakePromptBuilder) EvaluationPrompt(j Job) string { return "evaluate:" + j.ID }

func jsonProvider(text string) *modelprovider.GeminiProvider {
	return &modelprovider.GeminiProvider{GenerateFunc: func(ctx context.Context, req modelprovider.Request) (modelprovider.Response, error) {
		return modelprovider.Response{Text: text}, nil
	}}
}

func TestPipeline_Process_TriageRejectSkipsDeepEval(t *testing.T) {
	jobStore := newFakeStore()
	evalStore := newFakeEvalStore()
	triager := NewTriager(newTestRecorder(), jsonProvider(`{"fit_score": 20, "hard_fail": true, "recommendation": "reject"}`))
	evaluator := NewEvaluator(newTestRecorder(), jsonProvider(`{"score": 20, "recommendation": "reject"}`))

	p := NewPipeline(jobStore, evalStore, fakePromptBuilder{}, triager, evaluator, Limits{JobsDiscoveredPerDay: 100, DeepEvaluationsPerDay: 30}, idSeqJob("eval-1"))

	j := Job{ID: "job-1"}
	score, err := p.Process(context.Background(), j)
	if err != nil {
		t.Fatalf("Process() error: %v", err)
	}
	if score.Recommendation != RecommendationReject {
		t.Errorf("Recommendation = %s, want REJECT", score.Recommendation)
	}
	if score.EvaluatedBy != "triage" {
		t.Errorf("EvaluatedBy = %s, want triage (deep eval should be skipped)", score.EvaluatedBy)
	}
	if evalStore.deepCount != 0 {
		t.Errorf("deepCount = %d, want 0", evalStore.deepCount)
	}
}

func TestPipeline_Process_TriageApplyRunsDeepEval(t *testing.T) {
	jobStore := newFakeStore()
	evalStore := newFakeEvalStore()
	triager := NewTriager(newTestRecorder(), jsonProvider(`{"fit_score": 85, "recommendation": "apply"}`))
	evaluator := NewEvaluator(newTestRecorder(), jsonProvider(`{"score": 87, "recommendation": "apply", "confidence": "high"}`))

	p := NewPipeline(jobStore, evalStore, fakePromptBuilder{}, triager, evaluator, Limits{JobsDiscoveredPerDay: 100, DeepEvaluationsPerDay: 30}, idSeqJob("eval-1", "eval-2"))

	j := Job{ID: "job-1"}
	score, err := p.Process(context.Background(), j)
	if err != nil {
		t.Fatalf("Process() error: %v", err)
	}
	if score.EvaluatedBy != "deep_evaluation" {
		t.Errorf("EvaluatedBy = %s, want deep_evaluation", score.EvaluatedBy)
	}
	if score.Value != 87 {
		t.Errorf("Value = %d, want 87", score.Value)
	}
}

func TestPipeline_Process_DeepEvalLimitReached_FallsBackToTriageScore(t *testing.T) {
	jobStore := newFakeStore()
	evalStore := newFakeEvalStore()
	evalStore.deepCount = 30 // at limit
	triager := NewTriager(newTestRecorder(), jsonProvider(`{"fit_score": 85, "recommendation": "apply"}`))
	evaluator := NewEvaluator(newTestRecorder(), jsonProvider(`{"score": 87, "recommendation": "apply"}`))

	p := NewPipeline(jobStore, evalStore, fakePromptBuilder{}, triager, evaluator, Limits{JobsDiscoveredPerDay: 100, DeepEvaluationsPerDay: 30}, idSeqJob("eval-1"))

	j := Job{ID: "job-1"}
	score, err := p.Process(context.Background(), j)
	if err != nil {
		t.Fatalf("Process() error: %v", err)
	}
	if score.EvaluatedBy != "triage" {
		t.Errorf("EvaluatedBy = %s, want triage (deep-eval limit reached)", score.EvaluatedBy)
	}
}

func TestPipeline_Process_DiscoveryLimitReached(t *testing.T) {
	jobStore := newFakeStore()
	for i := 0; i < 5; i++ {
		jobStore.saved = append(jobStore.saved, Job{ID: "existing", DiscoveredAt: time.Now()})
	}
	evalStore := newFakeEvalStore()
	triager := NewTriager(newTestRecorder(), jsonProvider(`{"fit_score": 85, "recommendation": "apply"}`))
	evaluator := NewEvaluator(newTestRecorder(), jsonProvider(`{"score": 87, "recommendation": "apply"}`))

	p := NewPipeline(jobStore, evalStore, fakePromptBuilder{}, triager, evaluator, Limits{JobsDiscoveredPerDay: 5, DeepEvaluationsPerDay: 30}, idSeqJob("eval-1"))

	_, err := p.Process(context.Background(), Job{ID: "job-1"})
	if err == nil {
		t.Fatal("Process() expected error when discovery limit reached, got nil")
	}
}

func TestPipeline_Process_IdempotentTriage(t *testing.T) {
	jobStore := newFakeStore()
	evalStore := newFakeEvalStore()
	callCount := 0
	provider := &modelprovider.GeminiProvider{GenerateFunc: func(ctx context.Context, req modelprovider.Request) (modelprovider.Response, error) {
		callCount++
		return modelprovider.Response{Text: `{"fit_score": 85, "recommendation": "apply"}`}, nil
	}}
	triager := NewTriager(newTestRecorder(), provider)
	evaluator := NewEvaluator(newTestRecorder(), jsonProvider(`{"score": 87, "recommendation": "review"}`))

	p := NewPipeline(jobStore, evalStore, fakePromptBuilder{}, triager, evaluator, Limits{JobsDiscoveredPerDay: 100, DeepEvaluationsPerDay: 30}, idSeqJob("eval-1", "eval-2", "eval-3", "eval-4"))

	j := Job{ID: "job-1"}
	if _, err := p.Process(context.Background(), j); err != nil {
		t.Fatalf("first Process() error: %v", err)
	}
	if _, err := p.Process(context.Background(), j); err != nil {
		t.Fatalf("second Process() error: %v", err)
	}

	if callCount != 1 {
		t.Errorf("triage provider called %d times, want 1 (idempotent)", callCount)
	}
}

func idSeqJob(ids ...string) func() string {
	i := 0
	return func() string {
		id := ids[i%len(ids)]
		i++
		return id
	}
}
