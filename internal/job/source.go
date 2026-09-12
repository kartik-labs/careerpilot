package job

import "context"

// RawJob is what a Source returns before normalization — unprocessed
// fields as the source provided them.
type RawJob struct {
	SourceJobID string // empty if the source has no stable ID
	URL         string
	Company     string
	Title       string
	Location    string
	Description string
	Fields      map[string]string // any additional source-specific fields
}

// Source fetches job postings from a single external origin (job board,
// ATS, RSS feed, etc.). No concrete job-board API is integrated here —
// this repository has no credentials or API contract for one. Real
// integrations should implement this interface once a specific source's
// API is available; until then, MockSource covers ingestion/normalization
// testing.
type Source interface {
	// Name identifies the source (e.g. "greenhouse", "linkedin-manual-export").
	Name() string
	// FetchJobs returns newly available postings since the last call.
	// Sources are responsible for their own pagination/rate-limit handling.
	FetchJobs(ctx context.Context) ([]RawJob, error)
}

// MockSource is a deterministic, in-memory Source for tests and local
// development when no real job-source integration is configured.
type MockSource struct {
	SourceName string
	Jobs       []RawJob
	Err        error
}

func (m *MockSource) Name() string { return m.SourceName }

func (m *MockSource) FetchJobs(ctx context.Context) ([]RawJob, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return m.Jobs, nil
}
