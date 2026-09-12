# CareerPilot — Browser Recovery

## Objective

A browser interruption must never leave the user with an application that is impossible to resume.

## Session Lifecycle

```text
QUEUED
  |
STARTING
  |
RUNNING
  |
  +--> HUMAN_REQUIRED
  |       |
  |       +--> RESUMED
  |
  +--> COMPLETED
  |
  +--> FAILED
  |
  +--> EXPIRED
```

## Checkpoints

Persist after meaningful application steps:

```text
navigation
resume upload
form section
answer batch
review screen
before submission
```

Store:

```text
session_id
application_id
current_url
current_step
checkpoint
browser_context_reference
screenshot_reference
created_at
updated_at
expires_at
status
```

---

## CAPTCHA Handling

When a CAPTCHA is detected:

```text
1. Stop automation.
2. Persist browser state.
3. Capture screenshot.
4. Record current URL.
5. Create review request.
6. Generate temporary recovery link.
7. Notify user.
```

The system must not:

- solve CAPTCHA
- bypass CAPTCHA
- rotate IPs to evade detection
- repeatedly reload the page
- spawn parallel browser sessions

---

## Recovery Link

Example:

```text
https://careerpilot.example/recover/<opaque-token>
```

The token should not expose:

- cookies
- passwords
- browser storage
- API credentials

The recovery endpoint authenticates the user and maps the capability token to the stored browser session.

---

## Resume

After human completion:

```text
POST /api/browser/sessions/{id}/resume
```

Runner:

```text
restore context
verify expected page
continue from checkpoint
```

If the page has materially changed:

```text
HUMAN_REQUIRED
```

Do not guess.

---

## Timeouts

Every session has an expiration.

When expired:

```text
EXPIRED
```

Keep the application state for investigation, but do not continue browser activity automatically.

---

## Browser Crash

On crash:

```text
detect
  |
restore checkpoint
  |
verify page
  |
resume
```

If verification fails:

```text
HUMAN_REQUIRED
```
