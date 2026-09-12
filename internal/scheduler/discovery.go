package scheduler

import (
	"context"
	"fmt"

	"github.com/kartik-labs/careerpilot/internal/job"
)

// DiscoveryTask runs job discovery across configured sources: fetch raw
// jobs, ingest (normalize + dedupe), and persist. It is invoked once per
// scheduler trigger (e.g. Cloudflare Worker cron) — it does not loop or
// poll internally.
type DiscoveryTask struct {
	Sources []job.Source
	Store   job.Store
}

// DiscoveryResult summarizes one discovery run for audit reporting.
type DiscoveryResult struct {
	Fetched    int
	Ingested   int
	Duplicates int
}

// Run fetches jobs from every configured source and ingests them. A
// per-source fetch failure is recorded but does not abort the other
// sources — one dead job board must not block discovery entirely.
func (t *DiscoveryTask) Run(ctx context.Context) (DiscoveryResult, error) {
	var result DiscoveryResult
	var firstErr error

	for _, source := range t.Sources {
		raws, err := source.FetchJobs(ctx)
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("scheduler: source %s: %w", source.Name(), err)
			}
			continue
		}

		result.Fetched += len(raws)

		for _, raw := range raws {
			j, err := job.Ingest(t.Store, source.Name(), raw)
			if err != nil {
				if firstErr == nil {
					firstErr = fmt.Errorf("scheduler: ingest from %s: %w", source.Name(), err)
				}
				continue
			}

			if j.Status == job.StatusDuplicate {
				result.Duplicates++
			}

			if _, err := t.Store.SaveJob(j); err != nil {
				if firstErr == nil {
					firstErr = fmt.Errorf("scheduler: save job from %s: %w", source.Name(), err)
				}
				continue
			}
			result.Ingested++
		}
	}

	return result, firstErr
}
