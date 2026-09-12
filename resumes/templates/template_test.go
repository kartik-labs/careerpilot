package templates

import (
	"os"
	"strings"
	"testing"

	"github.com/kartik-labs/careerpilot/internal/resume/latex"
)

func TestBaseTemplate_RendersValidLatex(t *testing.T) {
	src, err := os.ReadFile("base.tex.tmpl")
	if err != nil {
		t.Fatalf("read template: %v", err)
	}

	out, err := latex.Render("base", string(src), latex.Data{
		FullName: "Ada Lovelace",
		Email:    "ada@example.com",
		Phone:    "+1-555-0100",
		Location: "Remote",
		Summary:  "Backend engineer.",
		Employment: []latex.EmploymentEntry{
			{
				Company:          "Analytical Engines Inc.",
				Title:            "Software Engineer",
				DateRange:        "2020 -- Present",
				Location:         "Remote",
				Responsibilities: []string{"Built distributed systems in Go."},
			},
		},
		Education: []latex.EducationEntry{
			{Institution: "Cambridge", Degree: "B.Sc.", FieldOfStudy: "Mathematics", DateRange: "2010 -- 2014"},
		},
		Skills: []string{"Go", "PostgreSQL"},
	})
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	if !strings.Contains(out, `\begin{document}`) || !strings.Contains(out, `\end{document}`) {
		t.Error("rendered output missing document boundaries")
	}
	if !strings.Contains(out, "Ada Lovelace") {
		t.Error("rendered output missing candidate name")
	}
	if strings.Contains(out, "<<") || strings.Contains(out, ">>") {
		t.Errorf("rendered output still contains unresolved template delimiters:\n%s", out)
	}
}
