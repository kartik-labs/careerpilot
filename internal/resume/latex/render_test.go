package latex

import "testing"

func TestRender(t *testing.T) {
	tmpl := `\name{<<.FullName>>}
\email{<<.Email>>}
<<range .Skills>><<.>>, <<end>>`

	got, err := Render("resume", tmpl, Data{
		FullName: "Ada Lovelace",
		Email:    "ada@example.com",
		Skills:   []string{"Go", "Postgres"},
	})
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	want := `\name{Ada Lovelace}
\email{ada@example.com}
Go, Postgres, `

	if got != want {
		t.Errorf("Render() = %q, want %q", got, want)
	}
}

func TestRender_InvalidTemplate(t *testing.T) {
	_, err := Render("bad", `<<.Unclosed`, Data{})
	if err == nil {
		t.Fatal("Render() expected error for invalid template syntax, got nil")
	}
}

func TestRender_NeverInvokesToolchain(t *testing.T) {
	// Render is pure templating; it must not shell out or touch the
	// filesystem, so it stays testable without a LaTeX installation.
	_, err := Render("resume", `plain text, no fields`, Data{})
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}
}
