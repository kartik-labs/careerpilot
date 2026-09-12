package browser

import (
	"context"
	"testing"
)

func TestFakeDriver_NavigateReturnsScriptedStates(t *testing.T) {
	d := &FakeDriver{NavigateStates: []PageState{StateNormal, StateCaptcha}}
	ctx := context.Background()

	s1, err := d.Navigate(ctx, "https://example.com/apply")
	if err != nil {
		t.Fatalf("Navigate() error: %v", err)
	}
	if s1 != StateNormal {
		t.Errorf("first Navigate() = %s, want normal", s1)
	}

	s2, _ := d.Navigate(ctx, "https://example.com/apply/step2")
	if s2 != StateCaptcha {
		t.Errorf("second Navigate() = %s, want captcha", s2)
	}

	if len(d.NavigatedURLs) != 2 {
		t.Errorf("NavigatedURLs = %v, want 2 entries", d.NavigatedURLs)
	}
}

func TestFakeDriver_NavigateError(t *testing.T) {
	d := &FakeDriver{NavigateErr: context.DeadlineExceeded}
	_, err := d.Navigate(context.Background(), "https://example.com")
	if err == nil {
		t.Fatal("Navigate() expected error, got nil")
	}
}

func TestFakeDriver_FillRecordsFields(t *testing.T) {
	d := &FakeDriver{}
	fields := []FieldValue{{Selector: "#name", Value: "Ada"}}
	if err := d.Fill(context.Background(), fields); err != nil {
		t.Fatalf("Fill() error: %v", err)
	}
	if len(d.FilledFields) != 1 || d.FilledFields[0][0].Value != "Ada" {
		t.Errorf("FilledFields = %v, want recorded fill", d.FilledFields)
	}
}

func TestFakeDriver_Screenshot(t *testing.T) {
	d := &FakeDriver{}
	shot, err := d.Screenshot(context.Background())
	if err != nil {
		t.Fatalf("Screenshot() error: %v", err)
	}
	if len(shot.Data) == 0 {
		t.Error("expected non-empty screenshot data")
	}
}

func TestFakeDriver_Close(t *testing.T) {
	d := &FakeDriver{}
	if err := d.Close(context.Background()); err != nil {
		t.Fatalf("Close() error: %v", err)
	}
	if !d.Closed {
		t.Error("expected Closed = true")
	}
}
