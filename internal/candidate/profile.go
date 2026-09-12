// Package candidate defines the candidate profile domain model — the
// single source of truth for all facts used to generate resumes and
// application answers. See CLAUDE.md "Candidate Profile" and
// docs/IMPLEMENTATION-PLAN.md section 1.1.
package candidate

import "time"

// Profile is the source of truth for a candidate's career facts.
// Resumes and application answers are generated views derived from a
// specific Profile version; they must never introduce facts the profile
// does not contain.
type Profile struct {
	ID      string
	Version int

	Identity       ContactInfo
	Education      []Education
	Employment     []Employment
	Projects       []Project
	Technologies   []string
	Certifications []Certification
	Achievements   []Achievement

	Preferences Preferences

	CreatedAt time.Time
	UpdatedAt time.Time
}

// ContactInfo holds identity and contact details.
type ContactInfo struct {
	FullName string
	Email    string
	Phone    string
	Location string
	LinkedIn string
	GitHub   string
	Website  string
}

// Education is a single degree/program entry.
type Education struct {
	Institution  string
	Degree       string
	FieldOfStudy string
	StartDate    time.Time
	EndDate      time.Time // zero value means ongoing
	Notes        string
}

// Employment is a single job history entry.
type Employment struct {
	Company          string
	Title            string
	StartDate        time.Time
	EndDate          time.Time // zero value means current
	Location         string
	Responsibilities []string
	Technologies     []string
}

// Project is a notable project the candidate can reference.
type Project struct {
	Name         string
	Description  string
	Technologies []string
	URL          string
	StartDate    time.Time
	EndDate      time.Time
}

// Certification is a professional certification.
type Certification struct {
	Name         string
	Issuer       string
	IssuedDate   time.Time
	ExpiryDate   time.Time
	CredentialID string
}

// Achievement is a standalone accomplishment not tied to a specific job.
type Achievement struct {
	Description string
	Date        time.Time
}

// Preferences captures job search preferences and answers that are reused
// across applications.
type Preferences struct {
	TargetRoles       []string
	TargetLocations   []string
	RemotePolicy      string // e.g. "remote", "hybrid", "onsite", "flexible"
	WorkAuthorization WorkAuthorization
	Compensation      CompensationPreference
}

// WorkAuthorization records the candidate's authorization status per
// country/region. This must never be guessed or inferred — see
// docs/AGENT-POLICY.md "Forbidden".
type WorkAuthorization struct {
	Country             string
	Status              string // e.g. "citizen", "permanent_resident", "visa_required", "sponsorship_required"
	RequiresSponsorship bool
	Notes               string
}

// CompensationPreference records salary expectations, if the candidate has
// explicitly provided them. Zero values mean "not specified" — the answer
// engine must treat that as REVIEW, never guess a number.
type CompensationPreference struct {
	Currency      string
	MinimumAnnual int
	TargetAnnual  int
	Negotiable    bool
}
