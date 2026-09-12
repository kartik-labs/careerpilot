package latex

import (
	"context"
	"os/exec"
	"testing"
)

func TestNewLatexmkCompiler_Defaults(t *testing.T) {
	c := NewLatexmkCompiler()
	if c.binary() != "latexmk" {
		t.Errorf("binary() = %q, want latexmk", c.binary())
	}

	c.Binary = "custom-latexmk"
	if c.binary() != "custom-latexmk" {
		t.Errorf("binary() = %q, want custom-latexmk", c.binary())
	}
}

func TestLatexmkCompiler_Compile_RequiresToolchain(t *testing.T) {
	if _, err := exec.LookPath("latexmk"); err == nil {
		t.Skip("latexmk is installed; full compile is covered by an integration test environment with TeX Live")
	}

	c := NewLatexmkCompiler()
	_, err := c.Compile(context.Background(), `\documentclass{article}\begin{document}hi\end{document}`)
	if err == nil {
		t.Fatal("Compile() expected error when latexmk is not installed, got nil")
	}
}

func TestCountPDFPages_MissingPdfinfo(t *testing.T) {
	if _, err := exec.LookPath("pdfinfo"); err == nil {
		t.Skip("pdfinfo is installed; behavior covered by full compile integration test")
	}

	_, err := countPDFPages(context.Background(), "nonexistent.pdf")
	if err == nil {
		t.Fatal("countPDFPages() expected error when pdfinfo is not installed, got nil")
	}
}
