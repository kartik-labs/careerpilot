// Package notification delivers CareerPilot notifications to the user:
// application review requests and human-required browser handoffs.
// No concrete delivery channel (email, SMS, push) is wired up here — no
// credentials or provider contract were given, so this package defines
// the interface plus a log-backed reference implementation, per the
// project's rule against inventing external integrations.
package notification

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// Kind classifies a notification for filtering/routing.
type Kind string

const (
	KindReviewRequired Kind = "application.review_required"
	KindHumanRequired  Kind = "application.human_required"
)

// Notification is a single message queued for delivery to the user. The
// message template follows docs/AGENT-POLICY.md "Human Review Message":
// company, role, reason, current progress, what the user needs to do, a
// session/resume link if applicable, and an expiration time — never a
// vague "something went wrong."
type Notification struct {
	ID   string
	Kind Kind

	Company  string
	Role     string
	Reason   string
	Progress string // e.g. "step 6/8" or "resume prepared, awaiting answers"

	ActionURL string // recovery link or review link, if applicable
	ExpiresAt time.Time

	CreatedAt time.Time
}

// Sender delivers a Notification through some concrete channel.
// internal/browser.Notifier is a narrower interface (CAPTCHA/human-required
// handoff only); Sender implementations can satisfy both.
type Sender interface {
	Send(ctx context.Context, n Notification) error
}

// Store persists notifications for audit/history
// (docs/IMPLEMENTATION-PLAN.md "notifications" table).
type Store interface {
	SaveNotification(n Notification) error
}

// Service is the entry point business logic uses to raise notifications.
// It always persists before/regardless of delivery outcome, so a failed
// send never erases the audit trail.
type Service struct {
	Sender Sender
	Store  Store
	NewID  func() string
	Now    func() time.Time
}

// NewService builds a Service.
func NewService(sender Sender, store Store, newID func() string) *Service {
	return &Service{Sender: sender, Store: store, NewID: newID, Now: time.Now}
}

// NotifyReviewRequired raises an application.review_required notification
// (e.g. a resume awaiting approval, or an answer needing human input).
func (s *Service) NotifyReviewRequired(ctx context.Context, company, role, reason, progress, actionURL string) error {
	return s.notify(ctx, Notification{
		Kind:      KindReviewRequired,
		Company:   company,
		Role:      role,
		Reason:    reason,
		Progress:  progress,
		ActionURL: actionURL,
	})
}

// NotifyHumanRequired raises an application.human_required notification.
// This method's signature matches internal/browser.Notifier so *Service
// can be passed directly as a browser.Runner's Notifier.
func (s *Service) NotifyHumanRequired(ctx context.Context, sessionID, applicationID, reason, recoveryURL string) error {
	return s.notify(ctx, Notification{
		Kind:      KindHumanRequired,
		Reason:    reason,
		Progress:  fmt.Sprintf("session %s (application %s)", sessionID, applicationID),
		ActionURL: recoveryURL,
	})
}

func (s *Service) notify(ctx context.Context, n Notification) error {
	n.ID = s.NewID()
	n.CreatedAt = s.Now()

	if err := s.Store.SaveNotification(n); err != nil {
		return fmt.Errorf("notification: save: %w", err)
	}

	if err := s.Sender.Send(ctx, n); err != nil {
		return fmt.Errorf("notification: send: %w", err)
	}

	return nil
}

// LogSender is a reference Sender implementation that logs notifications
// via structured logging. Suitable for local development and as the
// default until a real channel (email/Slack/push) is configured.
type LogSender struct {
	Logger *slog.Logger
}

func (s *LogSender) Send(ctx context.Context, n Notification) error {
	s.Logger.InfoContext(ctx, "notification",
		"kind", n.Kind,
		"company", n.Company,
		"role", n.Role,
		"reason", n.Reason,
		"progress", n.Progress,
		"action_url", n.ActionURL,
	)
	return nil
}
