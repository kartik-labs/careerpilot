package job

import "fmt"

// Deduplicate checks whether a normalized Job already exists in store,
// preferring stable source identity over the fingerprint fallback
// (docs/IMPLEMENTATION-PLAN.md "deduplication using stable job identity
// where available and deterministic fallback fingerprints otherwise").
//
// It returns the existing Job and true if j is a duplicate; store lookups
// are read-only, no mutation happens here.
func Deduplicate(store Store, j Job) (existing Job, isDuplicate bool, err error) {
	if j.SourceJobID != "" {
		found, ok, err := store.FindBySourceJobID(j.Source, j.SourceJobID)
		if err != nil {
			return Job{}, false, fmt.Errorf("job: dedup by source id: %w", err)
		}
		if ok {
			return found, true, nil
		}
	}

	found, ok, err := store.FindByNormalizedHash(j.NormalizedHash)
	if err != nil {
		return Job{}, false, fmt.Errorf("job: dedup by fingerprint: %w", err)
	}
	if ok {
		return found, true, nil
	}

	return Job{}, false, nil
}

// Ingest normalizes a raw job, checks for duplicates, and returns the Job
// ready to be saved (Status DUPLICATE with DuplicateOfID set, or
// DISCOVERED->NORMALIZED for a genuinely new posting). It does not itself
// call store.SaveJob — callers control the persistence boundary and
// idempotency key.
func Ingest(store Store, source string, raw RawJob) (Job, error) {
	j := Normalize(source, raw)

	existing, dup, err := Deduplicate(store, j)
	if err != nil {
		return Job{}, err
	}
	if dup {
		j.Status = StatusDuplicate
		j.DuplicateOfID = existing.ID
	}

	return j, nil
}
