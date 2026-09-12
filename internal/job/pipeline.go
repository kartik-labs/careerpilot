package job

import (
	"context"
	"fmt"
	"time"
)

// Limits are the daily caps this pipeline must respect
// (CLAUDE.md "Initial Resource Limits").
type Limits struct {
	JobsDiscoveredPerDay  int
	DeepEvaluationsPerDay int
}

// PromptBuilder renders the triage/evaluation prompt text for a job. Kept
// as an interface so prompt templates (candidate profile facts, job
// description, etc.) stay outside this orchestration package.
type PromptBuilder interface {
	TriagePrompt(j Job) string
	EvaluationPrompt(j Job) string
}

// EvaluationStore persists per-job evaluation history
// (docs/IMPLEMENTATION-PLAN.md section 10 "job_evaluations").
type EvaluationStore interface {
	SaveEvaluation(e Evaluation) error
	CountDeepEvaluationsSince(since time.Time) (int, error)
	// GetEvaluation supports idempotency: if a job was already evaluated at
	// this stage, Pipeline returns the existing Score instead of
	// re-invoking the model.
	GetEvaluation(jobID, stage string) (Evaluation, bool, error)
}

// Evaluation is a persisted audit record of one scoring stage.
type Evaluation struct {
	ID        string
	JobID     string
	Stage     string // "triage" | "deep_evaluation"
	Score     Score
	CreatedAt time.Time
}

// Pipeline orchestrates job intelligence: triage -> hard filter -> deep
// evaluation, enforcing daily limits and idempotency. It never mutates
// Job/Score state based on raw model output — ApplyTriage/ApplyEvaluation
// do that deterministically before anything is persisted.
type Pipeline struct {
	JobStore  Store
	EvalStore EvaluationStore
	Prompts   PromptBuilder
	Triager   *Triager
	Evaluator *Evaluator
	Limits    Limits
	NewID     func() string
	Now       func() time.Time
}

// NewPipeline builds a Pipeline.
func NewPipeline(jobStore Store, evalStore EvaluationStore, prompts PromptBuilder, triager *Triager, evaluator *Evaluator, limits Limits, newID func() string) *Pipeline {
	return &Pipeline{
		JobStore:  jobStore,
		EvalStore: evalStore,
		Prompts:   prompts,
		Triager:   triager,
		Evaluator: evaluator,
		Limits:    limits,
		NewID:     newID,
		Now:       time.Now,
	}
}

// Process runs one job through triage, and if it passes, through deep
// evaluation. It returns the final Score plus a bool indicating whether
// deep evaluation actually ran.
func (p *Pipeline) Process(ctx context.Context, j Job) (Score, error) {
	startOfDay := p.startOfToday()

	discoveredCount, err := p.JobStore.CountDiscoveredSince(startOfDay)
	if err != nil {
		return Score{}, fmt.Errorf("job: check discovery limit: %w", err)
	}
	if discoveredCount >= p.Limits.JobsDiscoveredPerDay {
		return Score{}, fmt.Errorf("job: daily discovery limit (%d) reached", p.Limits.JobsDiscoveredPerDay)
	}

	triageScore, err := p.runTriage(ctx, j)
	if err != nil {
		return Score{}, err
	}

	if triageScore.Recommendation != RecommendationApply {
		return triageScore, nil
	}

	deepCount, err := p.EvalStore.CountDeepEvaluationsSince(startOfDay)
	if err != nil {
		return Score{}, fmt.Errorf("job: check deep-eval limit: %w", err)
	}
	if deepCount >= p.Limits.DeepEvaluationsPerDay {
		// Daily deep-evaluation budget exhausted: fall back to the triage
		// score rather than escalating past a configured limit.
		return triageScore, nil
	}

	return p.runDeepEvaluation(ctx, j)
}

func (p *Pipeline) runTriage(ctx context.Context, j Job) (Score, error) {
	existing, done, err := p.EvalStore.GetEvaluation(j.ID, "triage")
	if err != nil {
		return Score{}, fmt.Errorf("job: check triage idempotency: %w", err)
	}
	if done {
		return existing.Score, nil
	}

	prompt := p.Prompts.TriagePrompt(j)
	out, err := p.Triager.Triage(ctx, j.ID, prompt)
	if err != nil {
		return Score{}, err
	}

	score := ApplyTriage(out)
	if err := p.EvalStore.SaveEvaluation(Evaluation{
		ID:        p.NewID(),
		JobID:     j.ID,
		Stage:     "triage",
		Score:     score,
		CreatedAt: p.Now(),
	}); err != nil {
		return Score{}, fmt.Errorf("job: save triage evaluation: %w", err)
	}

	return score, nil
}

func (p *Pipeline) runDeepEvaluation(ctx context.Context, j Job) (Score, error) {
	existing, done, err := p.EvalStore.GetEvaluation(j.ID, "deep_evaluation")
	if err != nil {
		return Score{}, fmt.Errorf("job: check deep-evaluation idempotency: %w", err)
	}
	if done {
		return existing.Score, nil
	}

	prompt := p.Prompts.EvaluationPrompt(j)
	out, err := p.Evaluator.Evaluate(ctx, j.ID, prompt)
	if err != nil {
		return Score{}, err
	}

	score := ApplyEvaluation(out)
	if err := p.EvalStore.SaveEvaluation(Evaluation{
		ID:        p.NewID(),
		JobID:     j.ID,
		Stage:     "deep_evaluation",
		Score:     score,
		CreatedAt: p.Now(),
	}); err != nil {
		return Score{}, fmt.Errorf("job: save deep evaluation: %w", err)
	}

	return score, nil
}

func (p *Pipeline) startOfToday() time.Time {
	now := p.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
}
