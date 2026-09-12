package application

import "fmt"

// CheckDuplicate reports whether an application already exists for jobID.
// Duplicate detection is keyed on job identity, since CareerPilot's rule
// is one application per job per candidate profile
// (docs/IMPLEMENTATION-PLAN.md "applications.job_id" index;
// CLAUDE.md "Application Safety Gate": "duplicate application status").
func CheckDuplicate(store Store, jobID string) (existing Application, isDuplicate bool, err error) {
	found, ok, err := store.FindByJobID(jobID)
	if err != nil {
		return Application{}, false, fmt.Errorf("application: check duplicate: %w", err)
	}
	return found, ok, nil
}
