package application

import "time"

// EventType names a recorded application lifecycle event
// (docs/IMPLEMENTATION-PLAN.md "application_events";
// CLAUDE.md "Event Model").
type EventType string

const (
	EventCreated          EventType = "application.created"
	EventQualified        EventType = "application.qualified"
	EventPreparingStarted EventType = "application.preparing_started"
	EventResumeAttached   EventType = "application.resume_attached"
	EventReviewRequired   EventType = "application.review_required"
	EventReadyForSubmit   EventType = "application.ready"
	EventDuplicateBlocked EventType = "application.duplicate_blocked"
	EventCancelled        EventType = "application.cancelled"
	EventSubmitted        EventType = "application.submitted"
)

// Event is a single immutable audit record. Every important state
// transition must produce one (CLAUDE.md "Every important state
// transition should be auditable.").
type Event struct {
	ID            string
	ApplicationID string
	JobID         string
	Type          EventType
	Detail        string
	CreatedAt     time.Time
}

// AuditStore persists Events. Handlers appending events must be
// idempotent (CLAUDE.md "Event Model": "Handlers should be idempotent.") —
// callers are expected to derive Event.ID deterministically (e.g. from
// applicationID+eventType+a monotonic version) so replays don't duplicate
// records; this package does not enforce that itself.
type AuditStore interface {
	AppendEvent(e Event) error
	ListEvents(applicationID string) ([]Event, error)
}

// RecordEvent builds and appends an Event for an application transition.
func RecordEvent(store AuditStore, newID func() string, now func() time.Time, a Application, eventType EventType, detail string) error {
	return store.AppendEvent(Event{
		ID:            newID(),
		ApplicationID: a.ID,
		JobID:         a.JobID,
		Type:          eventType,
		Detail:        detail,
		CreatedAt:     now(),
	})
}
