// Package latex renders resume content into LaTeX source and compiles it
// to PDF. Rendering is deterministic Go templating; compilation shells out
// to a LaTeX toolchain (latexmk) so business logic never depends on a
// specific PDF library. See docs/IMPLEMENTATION-PLAN.md section 1.4.
package latex

import (
	"bytes"
	"fmt"
	"text/template"
)

// Data is the set of fields a resume LaTeX template can reference. All
// values must already be resolved, factual claim text — no placeholder
// content is invented at render time.
type Data struct {
	FullName string
	Email    string
	Phone    string
	Location string
	LinkedIn string
	GitHub   string

	Summary string

	Employment []EmploymentEntry
	Education  []EducationEntry
	Projects   []ProjectEntry
	Skills     []string
}

// EmploymentEntry is a single rendered job history block.
type EmploymentEntry struct {
	Company          string
	Title            string
	DateRange        string
	Location         string
	Responsibilities []string
}

// EducationEntry is a single rendered education block.
type EducationEntry struct {
	Institution  string
	Degree       string
	FieldOfStudy string
	DateRange    string
}

// ProjectEntry is a single rendered project block.
type ProjectEntry struct {
	Name         string
	Description  string
	Technologies []string
}

// Render executes the named template against data and returns LaTeX
// source. templateSource is the raw .tex template content (typically read
// from the resumes/ directory by the caller).
func Render(name, templateSource string, data Data) (string, error) {
	tmpl, err := template.New(name).Delims("<<", ">>").Parse(templateSource)
	if err != nil {
		return "", fmt.Errorf("latex: parse template %s: %w", name, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("latex: execute template %s: %w", name, err)
	}

	return buf.String(), nil
}
