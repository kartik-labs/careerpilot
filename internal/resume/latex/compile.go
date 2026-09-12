package latex

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// CompileResult is the raw output of a LaTeX compilation attempt.
type CompileResult struct {
	PDF       []byte
	PageCount int
}

// Compiler compiles LaTeX source to PDF and reports its page count.
// Business logic (internal/resume) depends on this interface so it never
// couples to a specific LaTeX distribution or shell invocation.
type Compiler interface {
	Compile(ctx context.Context, texSource string) (CompileResult, error)
}

// LatexmkCompiler compiles using an external `latexmk` binary, which must
// be present on PATH (e.g. via a TeX Live installation on the on-demand
// runner or CI image). It never modifies texSource to force a page count.
type LatexmkCompiler struct {
	// Binary is the latexmk executable name/path. Defaults to "latexmk".
	Binary string
}

// NewLatexmkCompiler returns a Compiler backed by the latexmk binary.
func NewLatexmkCompiler() *LatexmkCompiler {
	return &LatexmkCompiler{Binary: "latexmk"}
}

func (c *LatexmkCompiler) binary() string {
	if c.Binary != "" {
		return c.Binary
	}
	return "latexmk"
}

// Compile writes texSource to a temp directory, runs latexmk to produce a
// PDF, and counts its pages.
func (c *LatexmkCompiler) Compile(ctx context.Context, texSource string) (CompileResult, error) {
	dir, err := os.MkdirTemp("", "careerpilot-resume-*")
	if err != nil {
		return CompileResult{}, fmt.Errorf("latex: create temp dir: %w", err)
	}
	defer os.RemoveAll(dir)

	texPath := filepath.Join(dir, "resume.tex")
	if err := os.WriteFile(texPath, []byte(texSource), 0o600); err != nil {
		return CompileResult{}, fmt.Errorf("latex: write source: %w", err)
	}

	cmd := exec.CommandContext(ctx, c.binary(), "-interaction=nonstopmode", "-halt-on-error", "-pdf", "-outdir="+dir, texPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return CompileResult{}, fmt.Errorf("latex: compile failed: %w\n%s", err, output)
	}

	pdfPath := filepath.Join(dir, "resume.pdf")
	pdf, err := os.ReadFile(pdfPath)
	if err != nil {
		return CompileResult{}, fmt.Errorf("latex: read compiled pdf: %w", err)
	}

	pageCount, err := countPDFPages(ctx, pdfPath)
	if err != nil {
		return CompileResult{}, fmt.Errorf("latex: count pages: %w", err)
	}

	return CompileResult{PDF: pdf, PageCount: pageCount}, nil
}

// countPDFPages shells out to `pdfinfo` (poppler-utils), which correctly
// handles compressed object streams that a naive byte scan of the PDF
// would miscount. Given the one-page requirement is a hard gate, an
// unreliable count is worse than a hard dependency on a well-established
// tool — install poppler-utils alongside the LaTeX toolchain.
func countPDFPages(ctx context.Context, pdfPath string) (int, error) {
	cmd := exec.CommandContext(ctx, "pdfinfo", pdfPath)
	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("run pdfinfo: %w", err)
	}

	for _, line := range strings.Split(string(output), "\n") {
		if after, ok := strings.CutPrefix(line, "Pages:"); ok {
			n, err := strconv.Atoi(strings.TrimSpace(after))
			if err != nil {
				return 0, fmt.Errorf("parse pdfinfo Pages line %q: %w", line, err)
			}
			return n, nil
		}
	}

	return 0, fmt.Errorf("pdfinfo output missing Pages line")
}
