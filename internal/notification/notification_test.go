package notification

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/kartik-labs/careerpilot/internal/browser"
)

type fakeSender struct {
	sent []Notification
	err  error
}

func (s *fakeSender) Send(ctx context.Context, n Notification) error {
	s.sent = append(s.sent, n)
	return s.err
}

type fakeStore struct {
	saved []Notification
	err   error
}

func (s *fakeStore) SaveNotification(n Notification) error {
	if s.err != nil {
		return s.err
	}
	s.saved = append(s.saved, n)
	return nil
}

func testService(sender *fakeSender, store *fakeStore) *Service {
	i := 0
	svc := NewService(sender, store, func() string {
		i++
		return "notif-" + string(rune('0'+i))
	})
	svc.Now = func() time.Time { return time.Unix(0, 0) }
	return svc
}

func TestNotifyReviewRequired(t *testing.T) {
	sender := &fakeSender{}
	store := &fakeStore{}
	svc := testService(sender, store)

	err := svc.NotifyReviewRequired(context.Background(), "Acme", "Backend Engineer", "resume awaiting approval", "step 3/5", "/review/abc")
	if err != nil {
		t.Fatalf("NotifyReviewRequired() error: %v", err)
	}
	if len(sender.sent) != 1 {
		t.Fatalf("sent = %d, want 1", len(sender.sent))
	}
	if sender.sent[0].Kind != KindReviewRequired {
		t.Errorf("Kind = %s, want %s", sender.sent[0].Kind, KindReviewRequired)
	}
	if len(store.saved) != 1 {
		t.Errorf("saved = %d, want 1", len(store.saved))
	}
}

func TestNotifyHumanRequired(t *testing.T) {
	sender := &fakeSender{}
	store := &fakeStore{}
	svc := testService(sender, store)

	err := svc.NotifyHumanRequired(context.Background(), "sess-1", "app-1", "CAPTCHA detected", "/recovery/xyz")
	if err != nil {
		t.Fatalf("NotifyHumanRequired() error: %v", err)
	}
	if sender.sent[0].Kind != KindHumanRequired {
		t.Errorf("Kind = %s, want %s", sender.sent[0].Kind, KindHumanRequired)
	}
	if sender.sent[0].ActionURL != "/recovery/xyz" {
		t.Errorf("ActionURL = %q, want /recovery/xyz", sender.sent[0].ActionURL)
	}
}

func TestNotify_PersistsBeforeSendFailure(t *testing.T) {
	sender := &fakeSender{err: context.DeadlineExceeded}
	store := &fakeStore{}
	svc := testService(sender, store)

	err := svc.NotifyReviewRequired(context.Background(), "Acme", "Engineer", "reason", "progress", "")
	if err == nil {
		t.Fatal("expected error when Send fails")
	}
	if len(store.saved) != 1 {
		t.Error("expected notification to be persisted even though Send failed")
	}
}

func TestNotify_StoreFailureAbortsBeforeSend(t *testing.T) {
	sender := &fakeSender{}
	store := &fakeStore{err: context.DeadlineExceeded}
	svc := testService(sender, store)

	err := svc.NotifyReviewRequired(context.Background(), "Acme", "Engineer", "reason", "progress", "")
	if err == nil {
		t.Fatal("expected error when store fails")
	}
	if len(sender.sent) != 0 {
		t.Error("expected Send to be skipped when persistence fails")
	}
}

func TestService_SatisfiesBrowserNotifierInterface(t *testing.T) {
	var _ browser.Notifier = (*Service)(nil)
}

func TestLogSender_Send(t *testing.T) {
	sender := &LogSender{Logger: slog.Default()}
	err := sender.Send(context.Background(), Notification{Kind: KindHumanRequired, Reason: "test"})
	if err != nil {
		t.Fatalf("Send() error: %v", err)
	}
}
