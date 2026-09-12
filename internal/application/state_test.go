package application

import (
	"errors"
	"testing"
)

func TestValidTransition(t *testing.T) {
	cases := []struct {
		from, to Status
		want     bool
	}{
		{StatusDiscovered, StatusQualified, true},
		{StatusQualified, StatusPreparing, true},
		{StatusPreparing, StatusReview, true},
		{StatusReview, StatusReady, true},
		{StatusReview, StatusCancelled, true},
		{StatusReady, StatusStarting, true},
		{StatusStarting, StatusNavigating, true},
		{StatusNavigating, StatusHumanReq, true},
		{StatusHumanReq, StatusFilling, true},
		{StatusSubmitting, StatusSubmitted, true},
		{StatusSubmitted, StatusCancelled, false},
		{StatusDiscovered, StatusSubmitted, false},
		{StatusFailed, StatusReady, false},
		{StatusCancelled, StatusReady, false},
	}

	for _, tc := range cases {
		if got := ValidTransition(tc.from, tc.to); got != tc.want {
			t.Errorf("ValidTransition(%s, %s) = %v, want %v", tc.from, tc.to, got, tc.want)
		}
	}
}

func TestTransition_Success(t *testing.T) {
	a := Application{Status: StatusDiscovered}
	got, err := Transition(a, StatusQualified)
	if err != nil {
		t.Fatalf("Transition() error: %v", err)
	}
	if got.Status != StatusQualified {
		t.Errorf("Status = %s, want QUALIFIED", got.Status)
	}
}

func TestTransition_Invalid(t *testing.T) {
	a := Application{Status: StatusDiscovered}
	_, err := Transition(a, StatusSubmitted)
	if err == nil {
		t.Fatal("Transition() expected error, got nil")
	}
	var target *InvalidTransitionError
	if !errors.As(err, &target) {
		t.Errorf("error type = %T, want *InvalidTransitionError", err)
	}
}
