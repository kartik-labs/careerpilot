// Package outcome implements CareerPilot Phase 6: outcome tracking and
// analytics. It records what happened to each application (response,
// interview, final outcome) against the context that produced it (resume
// variant/version, job score, source), and computes read-only analytics
// over that history. See CLAUDE.md "AI Model Strategy" and
// docs/ARCHITECTURE.md.
//
// This package never mutates a candidate profile or resume fact, and its
// analytics never trigger automatic changes — they are recommendations
// for a human to act on (Phase 6 prompt: "Changes to the candidate
// profile or resume require review.").
package outcome

import "time"

// FunnelStage marks where an application currently sits in the outcome
// funnel. Stages are additive/append-only history, not a single mutable
// status — an application can pass through several before reaching a
// terminal one.
type FunnelStage string

const (
	StageDiscovered FunnelStage = "discovered"
	StageQualified  FunnelStage = "qualified"
	StagePrepared   FunnelStage = "prepared"
	StageSubmitted  FunnelStage = "submitted"
	StageResponded  FunnelStage = "responded" // recruiter response received
	StageInterview  FunnelStage = "interview"
	StageOffer      FunnelStage = "offer"
	StageRejected   FunnelStage = "rejected"
	StageWithdrawn  FunnelStage = "withdrawn"
)

// Record is the outcome-tracking row for a single application, joining
// every dimension Phase 6 requires: role, company, resume variant and
// version, application source, job score, application date, response,
// interview, and final outcome.
type Record struct {
	ID            string
	ApplicationID string
	JobID         string

	Role    string
	Company string
	Source  string // job source, e.g. "greenhouse", "linkedin-manual-export"

	ResumeVariant string
	ResumeVersion int

	JobScore int // job.Score.Value at the time of application

	ApplicationDate time.Time

	// Stages records every funnel transition observed for this
	// application, in order, each with its own timestamp — this is what
	// funnel/rate analytics compute over.
	Stages []StageEvent

	FinalOutcome FunnelStage // zero value means still in progress
}

// StageEvent is one funnel-stage transition with its timestamp.
type StageEvent struct {
	Stage     FunnelStage
	Timestamp time.Time
}

// HasStage reports whether the record ever reached the given stage.
func (r Record) HasStage(stage FunnelStage) bool {
	for _, e := range r.Stages {
		if e.Stage == stage {
			return true
		}
	}
	return false
}

// Store persists outcome Records.
type Store interface {
	GetRecord(applicationID string) (Record, error)
	ListRecords(since time.Time) ([]Record, error)
	SaveRecord(r Record) (Record, error)
}

// AppendStage adds a new StageEvent to a Record and updates FinalOutcome
// if the stage is terminal. It never removes or rewrites prior stage
// history — outcome tracking is append-only, matching CLAUDE.md's
// auditability principle applied to outcomes.
func AppendStage(r Record, stage FunnelStage, at time.Time) Record {
	r.Stages = append(r.Stages, StageEvent{Stage: stage, Timestamp: at})

	switch stage {
	case StageOffer, StageRejected, StageWithdrawn:
		r.FinalOutcome = stage
	}

	return r
}
