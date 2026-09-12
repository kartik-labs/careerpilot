package resume

import (
	"context"
	"errors"
	"testing"

	"github.com/kartik-labs/careerpilot/internal/candidate"
	"github.com/kartik-labs/careerpilot/internal/resume/latex"
)

type fakeCompiler struct {
	pageCount int
	err       error
}

func (f *fakeCompiler) Compile(ctx context.Context, texSource string) (latex.CompileResult, error) {
	if f.err != nil {
		return latex.CompileResult{}, f.err
	}
	return latex.CompileResult{PDF: []byte("%PDF-fake"), PageCount: f.pageCount}, nil
}

func TestPipeline_Generate_OnePagePasses(t *testing.T) {
	p := NewPipeline(&fakeCompiler{pageCount: 1})

	v, err := p.Generate(context.Background(), GenerateInput{
		ProfileID:      "profile-1",
		ProfileVersion: 3,
		Variant:        VariantSoftwareEngineer,
		VersionNumber:  1,
		TemplateName:   "resume",
		TemplateSource: `\name{<<.FullName>>}`,
		Data:           latex.Data{FullName: "Ada Lovelace"},
	})
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	if v.Status != StatusValidated {
		t.Errorf("Status = %s, want %s", v.Status, StatusValidated)
	}
	if v.PageCount != 1 {
		t.Errorf("PageCount = %d, want 1", v.PageCount)
	}
	if v.Validation == nil || !v.Validation.Passed {
		t.Error("expected passing validation result")
	}
}

func TestPipeline_Generate_TwoPagesGoesToReview(t *testing.T) {
	p := NewPipeline(&fakeCompiler{pageCount: 2})

	v, err := p.Generate(context.Background(), GenerateInput{
		TemplateName:   "resume",
		TemplateSource: `content`,
		Data:           latex.Data{},
	})
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	if v.Status != StatusReview {
		t.Errorf("Status = %s, want %s", v.Status, StatusReview)
	}
	if v.Validation == nil || v.Validation.Passed {
		t.Error("expected failing validation result")
	}
	if len(v.Validation.Reasons) == 0 {
		t.Error("expected structured reasons for review")
	}
}

func TestPipeline_Generate_CompileError(t *testing.T) {
	p := NewPipeline(&fakeCompiler{err: errors.New("latexmk not found")})

	_, err := p.Generate(context.Background(), GenerateInput{
		TemplateName:   "resume",
		TemplateSource: `content`,
	})
	if err == nil {
		t.Fatal("Generate() expected error when compile fails, got nil")
	}
}

func TestPipeline_Generate_InvalidTemplate(t *testing.T) {
	p := NewPipeline(&fakeCompiler{pageCount: 1})

	_, err := p.Generate(context.Background(), GenerateInput{
		TemplateName:   "resume",
		TemplateSource: `<<.Unclosed`,
	})
	if err == nil {
		t.Fatal("Generate() expected error for invalid template, got nil")
	}
}

func TestApprove(t *testing.T) {
	v := Version{Status: StatusValidated}

	approved, err := Approve(v)
	if err != nil {
		t.Fatalf("Approve() error: %v", err)
	}
	if approved.Status != StatusApproved {
		t.Errorf("Status = %s, want %s", approved.Status, StatusApproved)
	}
	if approved.ApprovedAt.IsZero() {
		t.Error("expected ApprovedAt to be set")
	}
}

func TestApprove_InvalidTransition(t *testing.T) {
	v := Version{Status: StatusDraft}
	if _, err := Approve(v); err == nil {
		t.Fatal("Approve() expected error from DRAFT status, got nil")
	}
}

func TestReject(t *testing.T) {
	v := Version{Status: StatusReview}
	rejected, err := Reject(v)
	if err != nil {
		t.Fatalf("Reject() error: %v", err)
	}
	if rejected.Status != StatusRejected {
		t.Errorf("Status = %s, want %s", rejected.Status, StatusRejected)
	}
}

func TestValidateClaimProvenance(t *testing.T) {
	allowed := []candidate.Claim{
		{ID: "c1", AllowedForResume: true},
		{ID: "c2", AllowedForResume: false},
	}

	changes := []Change{
		{Description: "add bullet", ClaimIDs: []string{"c1"}},
		{Description: "add unverified bullet", ClaimIDs: []string{"c2", "c3"}},
	}

	missing := ValidateClaimProvenance(changes, allowed)

	if len(missing) != 2 {
		t.Fatalf("missing = %v, want 2 entries (c2, c3)", missing)
	}
	got := map[string]bool{missing[0]: true, missing[1]: true}
	if !got["c2"] || !got["c3"] {
		t.Errorf("missing = %v, want [c2 c3]", missing)
	}
}

func TestValidateClaimProvenance_AllAllowed(t *testing.T) {
	allowed := []candidate.Claim{{ID: "c1", AllowedForResume: true}}
	changes := []Change{{ClaimIDs: []string{"c1"}}}

	if missing := ValidateClaimProvenance(changes, allowed); len(missing) != 0 {
		t.Errorf("missing = %v, want empty", missing)
	}
}
