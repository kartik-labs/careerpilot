package job

import (
	"context"
	"fmt"
	"testing"
	"time"
)

type fakeStore struct {
	bySourceID map[string]Job // key: source+"|"+sourceJobID
	byHash     map[string]Job
	saved      []Job
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		bySourceID: map[string]Job{},
		byHash:     map[string]Job{},
	}
}

func (s *fakeStore) GetJob(id string) (Job, error) {
	for _, j := range s.saved {
		if j.ID == id {
			return j, nil
		}
	}
	return Job{}, nil
}

func (s *fakeStore) FindBySourceJobID(source, sourceJobID string) (Job, bool, error) {
	j, ok := s.bySourceID[source+"|"+sourceJobID]
	return j, ok, nil
}

func (s *fakeStore) FindByNormalizedHash(hash string) (Job, bool, error) {
	j, ok := s.byHash[hash]
	return j, ok, nil
}

func (s *fakeStore) SaveJob(j Job) (Job, error) {
	if j.ID == "" {
		j.ID = fmt.Sprintf("job-%d", len(s.saved)+1)
	}
	s.saved = append(s.saved, j)
	if j.SourceJobID != "" {
		s.bySourceID[j.Source+"|"+j.SourceJobID] = j
	}
	s.byHash[j.NormalizedHash] = j
	return j, nil
}

func (s *fakeStore) CountDiscoveredSince(since time.Time) (int, error) {
	count := 0
	for _, j := range s.saved {
		if !j.DiscoveredAt.Before(since) {
			count++
		}
	}
	return count, nil
}

func TestDeduplicate_BySourceJobID(t *testing.T) {
	store := newFakeStore()
	existing := Job{ID: "job-1", Source: "greenhouse", SourceJobID: "gh-1", NormalizedHash: "irrelevant"}
	store.SaveJob(existing)

	candidate := Job{Source: "greenhouse", SourceJobID: "gh-1", NormalizedHash: "different-hash"}

	found, isDup, err := Deduplicate(store, candidate)
	if err != nil {
		t.Fatalf("Deduplicate() error: %v", err)
	}
	if !isDup {
		t.Fatal("expected duplicate via source job ID")
	}
	if found.ID != "job-1" {
		t.Errorf("found.ID = %q, want job-1", found.ID)
	}
}

func TestDeduplicate_ByFingerprintFallback(t *testing.T) {
	store := newFakeStore()
	existing := Job{ID: "job-2", Source: "manual", NormalizedHash: "hash-abc"}
	store.SaveJob(existing)

	candidate := Job{Source: "manual", NormalizedHash: "hash-abc"} // no SourceJobID

	found, isDup, err := Deduplicate(store, candidate)
	if err != nil {
		t.Fatalf("Deduplicate() error: %v", err)
	}
	if !isDup {
		t.Fatal("expected duplicate via fingerprint fallback")
	}
	if found.ID != "job-2" {
		t.Errorf("found.ID = %q, want job-2", found.ID)
	}
}

func TestDeduplicate_NoMatch(t *testing.T) {
	store := newFakeStore()
	candidate := Job{Source: "manual", NormalizedHash: "hash-new"}

	_, isDup, err := Deduplicate(store, candidate)
	if err != nil {
		t.Fatalf("Deduplicate() error: %v", err)
	}
	if isDup {
		t.Error("expected no duplicate for unseen job")
	}
}

func TestIngest_NewJob(t *testing.T) {
	store := newFakeStore()
	raw := RawJob{Company: "Acme", Title: "Backend Engineer", Location: "Remote"}

	j, err := Ingest(store, "manual", raw)
	if err != nil {
		t.Fatalf("Ingest() error: %v", err)
	}
	if j.Status != StatusNormalized {
		t.Errorf("Status = %q, want NORMALIZED", j.Status)
	}
	if j.DuplicateOfID != "" {
		t.Errorf("DuplicateOfID = %q, want empty", j.DuplicateOfID)
	}
}

func TestIngest_DuplicateJob(t *testing.T) {
	store := newFakeStore()
	raw := RawJob{Company: "Acme", Title: "Backend Engineer", Location: "Remote"}

	first, _ := Ingest(store, "manual", raw)
	first.ID = "existing-1"
	store.SaveJob(first)

	second, err := Ingest(store, "manual", raw)
	if err != nil {
		t.Fatalf("Ingest() error: %v", err)
	}
	if second.Status != StatusDuplicate {
		t.Errorf("Status = %q, want DUPLICATE", second.Status)
	}
	if second.DuplicateOfID != "existing-1" {
		t.Errorf("DuplicateOfID = %q, want existing-1", second.DuplicateOfID)
	}
}

func TestMockSource(t *testing.T) {
	src := &MockSource{
		SourceName: "test-source",
		Jobs:       []RawJob{{Title: "Engineer"}},
	}

	if src.Name() != "test-source" {
		t.Errorf("Name() = %q, want test-source", src.Name())
	}

	jobs, err := src.FetchJobs(context.Background())
	if err != nil {
		t.Fatalf("FetchJobs() error: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("FetchJobs() returned %d jobs, want 1", len(jobs))
	}
}
