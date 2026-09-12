package application

import (
	"testing"
	"time"
)

type fakeAuditStore struct {
	events []Event
}

func (s *fakeAuditStore) AppendEvent(e Event) error {
	s.events = append(s.events, e)
	return nil
}

func (s *fakeAuditStore) ListEvents(applicationID string) ([]Event, error) {
	var out []Event
	for _, e := range s.events {
		if e.ApplicationID == applicationID {
			out = append(out, e)
		}
	}
	return out, nil
}

func TestRecordEvent(t *testing.T) {
	store := &fakeAuditStore{}
	newID, now := fixedClock()
	a := Application{ID: "app-1", JobID: "job-1"}

	err := RecordEvent(store, newID, now, a, EventQualified, "score 87, recommendation APPLY")
	if err != nil {
		t.Fatalf("RecordEvent() error: %v", err)
	}

	events, _ := store.ListEvents("app-1")
	if len(events) != 1 {
		t.Fatalf("events = %d, want 1", len(events))
	}
	if events[0].Type != EventQualified {
		t.Errorf("Type = %s, want %s", events[0].Type, EventQualified)
	}
	if events[0].CreatedAt != time.Unix(0, 0) {
		t.Errorf("CreatedAt = %v, want frozen clock value", events[0].CreatedAt)
	}
}
