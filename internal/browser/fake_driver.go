package browser

import "context"

// FakeDriver is a deterministic, in-memory Driver for tests and any
// environment where launching a real browser is impractical (no
// display, no network egress, sandboxed CI). It never talks to a real
// browser; behavior is entirely scripted by the test.
type FakeDriver struct {
	LaunchErr error

	// NavigateStates is consumed in order, one per Navigate call. When
	// exhausted, Navigate returns the last state again.
	NavigateStates []PageState
	NavigateErr    error
	navigateCalls  int

	DetectStates []PageState
	detectCalls  int

	FillErr error

	SubmitState PageState
	SubmitErr   error

	Screenshots     []Screenshot
	screenshotCalls int
	CurrentURLValue string
	Closed          bool
	FilledFields    [][]FieldValue
	NavigatedURLs   []string
}

func (f *FakeDriver) Launch(ctx context.Context) error {
	return f.LaunchErr
}

func (f *FakeDriver) Navigate(ctx context.Context, url string) (PageState, error) {
	f.NavigatedURLs = append(f.NavigatedURLs, url)
	f.CurrentURLValue = url
	if f.NavigateErr != nil {
		return StateNavigateErr, f.NavigateErr
	}
	if len(f.NavigateStates) == 0 {
		return StateUnknown, nil
	}
	idx := f.navigateCalls
	if idx >= len(f.NavigateStates) {
		idx = len(f.NavigateStates) - 1
	}
	f.navigateCalls++
	return f.NavigateStates[idx], nil
}

func (f *FakeDriver) Fill(ctx context.Context, fields []FieldValue) error {
	f.FilledFields = append(f.FilledFields, fields)
	return f.FillErr
}

func (f *FakeDriver) DetectState(ctx context.Context) (PageState, error) {
	if len(f.DetectStates) == 0 {
		return StateUnknown, nil
	}
	idx := f.detectCalls
	if idx >= len(f.DetectStates) {
		idx = len(f.DetectStates) - 1
	}
	f.detectCalls++
	return f.DetectStates[idx], nil
}

func (f *FakeDriver) Screenshot(ctx context.Context) (Screenshot, error) {
	if f.screenshotCalls < len(f.Screenshots) {
		s := f.Screenshots[f.screenshotCalls]
		f.screenshotCalls++
		return s, nil
	}
	f.screenshotCalls++
	return Screenshot{Data: []byte("fake-screenshot"), MimeType: "image/png"}, nil
}

func (f *FakeDriver) CurrentURL(ctx context.Context) (string, error) {
	return f.CurrentURLValue, nil
}

func (f *FakeDriver) Submit(ctx context.Context, selector string) (PageState, error) {
	if f.SubmitErr != nil {
		return StateUnknown, f.SubmitErr
	}
	return f.SubmitState, nil
}

func (f *FakeDriver) Close(ctx context.Context) error {
	f.Closed = true
	return nil
}
