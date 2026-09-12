package job

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// Normalize converts a RawJob from a given source into a canonical Job.
// It never drops the original data — RawData retains every input field
// for audit (docs/IMPLEMENTATION-PLAN.md section 4).
func Normalize(source string, raw RawJob) Job {
	rawData := map[string]string{
		"url":         raw.URL,
		"company":     raw.Company,
		"title":       raw.Title,
		"location":    raw.Location,
		"description": raw.Description,
	}
	for k, v := range raw.Fields {
		rawData[k] = v
	}

	company := normalizeText(raw.Company)
	title := normalizeText(raw.Title)
	location := normalizeText(raw.Location)

	j := Job{
		Source:       source,
		SourceJobID:  raw.SourceJobID,
		URL:          raw.URL,
		Company:      company,
		Title:        title,
		Location:     location,
		Description:  raw.Description,
		RemotePolicy: inferRemotePolicy(location, raw.Description),
		RawData:      rawData,
		Status:       StatusNormalized,
	}
	j.NormalizedHash = Fingerprint(source, company, title, location)

	return j
}

// normalizeText trims whitespace and collapses internal whitespace runs,
// without altering casing or content — this is deduplication support, not
// a content rewrite.
func normalizeText(s string) string {
	fields := strings.Fields(s)
	return strings.Join(fields, " ")
}

// Fingerprint deterministically derives a dedup key from normalized
// company/title/location when the source has no stable job ID. Identical
// inputs always produce the identical fingerprint (docs/IMPLEMENTATION-PLAN.md
// "deduplication using stable job identity where available and
// deterministic fallback fingerprints otherwise").
func Fingerprint(source, company, title, location string) string {
	key := strings.ToLower(strings.Join([]string{source, company, title, location}, "|"))
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

// inferRemotePolicy does a conservative, deterministic keyword check.
// It never guesses beyond what the text states — ambiguous cases return
// "unspecified" rather than a fabricated classification.
func inferRemotePolicy(location, description string) string {
	text := strings.ToLower(location + " " + description)
	switch {
	case strings.Contains(text, "fully remote") || strings.Contains(text, "100% remote"):
		return "remote"
	case strings.Contains(text, "hybrid"):
		return "hybrid"
	case strings.Contains(text, "remote"):
		return "remote"
	case strings.Contains(text, "on-site") || strings.Contains(text, "onsite") || strings.Contains(text, "on site"):
		return "onsite"
	default:
		return "unspecified"
	}
}
