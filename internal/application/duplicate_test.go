package application

import (
	"testing"
	"time"
)

type fakeAppStore struct {
	byJobID map[string]Application
	saved   []Application
}

func newFakeAppStore() *fakeAppStore {
	return &fakeAppStore{byJobID: map[string]Application{}}
}

func (s *fakeAppStore) GetApplication(id string) (Application, error) {
	for _, a := range s.saved {
		if a.ID == id {
			return a, nil
		}
	}
	return Application{}, nil
}

func (s *fakeAppStore) FindByJobID(jobID string) (Application, bool, error) {
	a, ok := s.byJobID[jobID]
	return a, ok, nil
}

func (s *fakeAppStore) SaveApplication(a Application) (Application, error) {
	s.saved = append(s.saved, a)
	s.byJobID[a.JobID] = a
	return a, nil
}

func (s *fakeAppStore) CountSubmittedSince(since time.Time) (int, error) {
	count := 0
	for _, a := range s.saved {
		if a.Status == StatusSubmitted {
			count++
		}
	}
	return count, nil
}

func TestCheckDuplicate_Found(t *testing.T) {
	store := newFakeAppStore()
	store.SaveApplication(Application{ID: "app-1", JobID: "job-1"})

	existing, isDup, err := CheckDuplicate(store, "job-1")
	if err != nil {
		t.Fatalf("CheckDuplicate() error: %v", err)
	}
	if !isDup {
		t.Fatal("expected duplicate")
	}
	if existing.ID != "app-1" {
		t.Errorf("existing.ID = %q, want app-1", existing.ID)
	}
}

func TestCheckDuplicate_NotFound(t *testing.T) {
	store := newFakeAppStore()

	_, isDup, err := CheckDuplicate(store, "job-unseen")
	if err != nil {
		t.Fatalf("CheckDuplicate() error: %v", err)
	}
	if isDup {
		t.Error("expected no duplicate for unseen job")
	}
}
