package scheduler

import (
	"context"
	"testing"
	"time"

	"github.com/kartik-labs/careerpilot/internal/job"
	"github.com/kartik-labs/careerpilot/internal/modelprovider"
)

type fakeLister struct {
	jobs []job.Job
}

func (l *fakeLister) ListPendingEvaluation(limit int) ([]job.Job, error) {
	if limit < len(l.jobs) {
		return l.jobs[:limit], nil
	}
	return l.jobs, nil
}

type fakeEvalStore struct {
	evaluations map[string]job.Evaluation
	deepCount   int
}

func newFakeEvalStore() *fakeEvalStore {
	return &fakeEvalStore{evaluations: map[string]job.Evaluation{}}
}
func (s *fakeEvalStore) SaveEvaluation(e job.Evaluation) error {
	s.evaluations[e.JobID+"|"+e.Stage] = e
	if e.Stage == "deep_evaluation" {
		s.deepCount++
	}
	return nil
}
func (s *fakeEvalStore) CountDeepEvaluationsSince(since time.Time) (int, error) {
	return s.deepCount, nil
}
func (s *fakeEvalStore) GetEvaluation(jobID, stage string) (job.Evaluation, bool, error) {
	e, ok := s.evaluations[jobID+"|"+stage]
	return e, ok, nil
}

type fakePrompts struct{}

func (fakePrompts) TriagePrompt(j job.Job) string     { return "triage" }
func (fakePrompts) EvaluationPrompt(j job.Job) string { return "evaluate" }

func fakeRecorder() *modelprovider.Recorder {
	return modelprovider.NewRecorder(noopRunStore{}, idGen("run-"))
}

type noopRunStore struct{}

func (noopRunStore) SaveRun(r modelprovider.Run) error { return nil }

func jsonGemini(text string) *modelprovider.GeminiProvider {
	return &modelprovider.GeminiProvider{GenerateFunc: func(ctx context.Context, req modelprovider.Request) (modelprovider.Response, error) {
		return modelprovider.Response{Text: text}, nil
	}}
}

func TestEvaluationTask_Run(t *testing.T) {
	lister := &fakeLister{jobs: []job.Job{{ID: "job-1"}, {ID: "job-2"}}}
	evalStore := newFakeEvalStore()
	jobStore := newFakeJobStore()

	triager := job.NewTriager(fakeRecorder(), jsonGemini(`{"fit_score": 20, "hard_fail": true, "recommendation": "reject"}`))
	evaluator := job.NewEvaluator(fakeRecorder(), jsonGemini(`{"score": 20, "recommendation": "reject"}`))
	pipeline := job.NewPipeline(jobStore, evalStore, fakePrompts{}, triager, evaluator, job.Limits{JobsDiscoveredPerDay: 100, DeepEvaluationsPerDay: 30}, idGen("eval-"))

	task := &EvaluationTask{Lister: lister, Pipeline: pipeline, BatchSize: 10}

	result, err := task.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	if result.Processed != 2 {
		t.Errorf("Processed = %d, want 2", result.Processed)
	}
	if result.Rejected != 2 {
		t.Errorf("Rejected = %d, want 2", result.Rejected)
	}
}

func TestEvaluationTask_Run_RespectsBatchSize(t *testing.T) {
	lister := &fakeLister{jobs: []job.Job{{ID: "job-1"}, {ID: "job-2"}, {ID: "job-3"}}}
	evalStore := newFakeEvalStore()
	jobStore := newFakeJobStore()

	triager := job.NewTriager(fakeRecorder(), jsonGemini(`{"fit_score": 20, "hard_fail": true, "recommendation": "reject"}`))
	evaluator := job.NewEvaluator(fakeRecorder(), jsonGemini(`{"score": 20, "recommendation": "reject"}`))
	pipeline := job.NewPipeline(jobStore, evalStore, fakePrompts{}, triager, evaluator, job.Limits{JobsDiscoveredPerDay: 100, DeepEvaluationsPerDay: 30}, idGen("eval-"))

	task := &EvaluationTask{Lister: lister, Pipeline: pipeline, BatchSize: 2}

	result, err := task.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	if result.Processed != 2 {
		t.Errorf("Processed = %d, want 2 (bounded by BatchSize)", result.Processed)
	}
}
