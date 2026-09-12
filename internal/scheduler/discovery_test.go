package scheduler

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kartik-labs/careerpilot/internal/job"
)

type fakeJobStore struct {
	bySourceID map[string]job.Job
	byHash     map[string]job.Job
	saved      []job.Job
}

func newFakeJobStore() *fakeJobStore {
	return &fakeJobStore{bySourceID: map[string]job.Job{}, byHash: map[string]job.Job{}}
}

func (s *fakeJobStore) GetJob(id string) (job.Job, error) { return job.Job{}, nil }
func (s *fakeJobStore) FindBySourceJobID(source, sourceJobID string) (job.Job, bool, error) {
	j, ok := s.bySourceID[source+"|"+sourceJobID]
	return j, ok, nil
}
func (s *fakeJobStore) FindByNormalizedHash(hash string) (job.Job, bool, error) {
	j, ok := s.byHash[hash]
	return j, ok, nil
}
func (s *fakeJobStore) SaveJob(j job.Job) (job.Job, error) {
	if j.ID == "" {
		j.ID = "job-saved"
	}
	s.saved = append(s.saved, j)
	s.byHash[j.NormalizedHash] = j
	if j.SourceJobID != "" {
		s.bySourceID[j.Source+"|"+j.SourceJobID] = j
	}
	return j, nil
}
func (s *fakeJobStore) CountDiscoveredSince(since time.Time) (int, error) { return len(s.saved), nil }

func TestDiscoveryTask_Run_IngestsFromMultipleSources(t *testing.T) {
	store := newFakeJobStore()
	task := &DiscoveryTask{
		Sources: []job.Source{
			&job.MockSource{SourceName: "source-a", Jobs: []job.RawJob{{Title: "Engineer A"}}},
			&job.MockSource{SourceName: "source-b", Jobs: []job.RawJob{{Title: "Engineer B"}}},
		},
		Store: store,
	}

	result, err := task.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	if result.Fetched != 2 || result.Ingested != 2 {
		t.Errorf("result = %+v, want Fetched=2 Ingested=2", result)
	}
}

func TestDiscoveryTask_Run_DetectsDuplicates(t *testing.T) {
	store := newFakeJobStore()
	// Pre-seed a job with the same fingerprint as the incoming one.
	existing := job.Normalize("source-a", job.RawJob{Company: "Acme", Title: "Engineer", Location: "Remote"})
	existing.ID = "existing-1"
	store.SaveJob(existing)

	task := &DiscoveryTask{
		Sources: []job.Source{
			&job.MockSource{SourceName: "source-a", Jobs: []job.RawJob{{Company: "Acme", Title: "Engineer", Location: "Remote"}}},
		},
		Store: store,
	}

	result, err := task.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	if result.Duplicates != 1 {
		t.Errorf("Duplicates = %d, want 1", result.Duplicates)
	}
}

func TestDiscoveryTask_Run_OneSourceFailureDoesNotBlockOthers(t *testing.T) {
	store := newFakeJobStore()
	task := &DiscoveryTask{
		Sources: []job.Source{
			&job.MockSource{SourceName: "broken", Err: errors.New("source down")},
			&job.MockSource{SourceName: "healthy", Jobs: []job.RawJob{{Title: "Engineer"}}},
		},
		Store: store,
	}

	result, err := task.Run(context.Background())
	if err == nil {
		t.Fatal("Run() expected error to be reported for the broken source")
	}
	if result.Ingested != 1 {
		t.Errorf("Ingested = %d, want 1 (healthy source still processed)", result.Ingested)
	}
}
