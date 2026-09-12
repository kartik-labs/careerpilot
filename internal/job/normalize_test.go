package job

import "testing"

func TestNormalize_PreservesRawData(t *testing.T) {
	raw := RawJob{
		SourceJobID: "gh-123",
		URL:         "https://example.com/jobs/123",
		Company:     "  Acme   Corp  ",
		Title:       "Backend  Engineer",
		Location:    "Remote",
		Description: "Fully remote role.",
	}

	j := Normalize("greenhouse", raw)

	if j.RawData["company"] != raw.Company {
		t.Errorf("RawData[company] = %q, want original %q", j.RawData["company"], raw.Company)
	}
	if j.Company != "Acme Corp" {
		t.Errorf("Company = %q, want normalized whitespace", j.Company)
	}
	if j.Title != "Backend Engineer" {
		t.Errorf("Title = %q, want normalized whitespace", j.Title)
	}
	if j.SourceJobID != "gh-123" {
		t.Errorf("SourceJobID = %q, want gh-123", j.SourceJobID)
	}
	if j.Status != StatusNormalized {
		t.Errorf("Status = %q, want NORMALIZED", j.Status)
	}
}

func TestNormalize_RemotePolicyInference(t *testing.T) {
	cases := []struct {
		name        string
		location    string
		description string
		want        string
	}{
		{"fully remote phrase", "", "This is a fully remote role.", "remote"},
		{"remote keyword", "Remote", "", "remote"},
		{"hybrid keyword", "Hybrid - NYC", "", "hybrid"},
		{"onsite keyword", "On-site, SF", "", "onsite"},
		{"no signal", "San Francisco", "Great team.", "unspecified"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			j := Normalize("test", RawJob{Location: tc.location, Description: tc.description})
			if j.RemotePolicy != tc.want {
				t.Errorf("RemotePolicy = %q, want %q", j.RemotePolicy, tc.want)
			}
		})
	}
}

func TestFingerprint_Deterministic(t *testing.T) {
	a := Fingerprint("greenhouse", "Acme Corp", "Backend Engineer", "Remote")
	b := Fingerprint("greenhouse", "Acme Corp", "Backend Engineer", "Remote")
	if a != b {
		t.Errorf("Fingerprint not deterministic: %q != %q", a, b)
	}
}

func TestFingerprint_CaseInsensitive(t *testing.T) {
	a := Fingerprint("greenhouse", "Acme Corp", "Backend Engineer", "Remote")
	b := Fingerprint("greenhouse", "acme corp", "backend engineer", "remote")
	if a != b {
		t.Errorf("Fingerprint should be case-insensitive: %q != %q", a, b)
	}
}

func TestFingerprint_DiffersOnDifferentInput(t *testing.T) {
	a := Fingerprint("greenhouse", "Acme Corp", "Backend Engineer", "Remote")
	b := Fingerprint("greenhouse", "Acme Corp", "Frontend Engineer", "Remote")
	if a == b {
		t.Error("expected different fingerprints for different titles")
	}
}
