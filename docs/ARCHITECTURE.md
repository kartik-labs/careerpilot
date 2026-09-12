# CareerPilot — Architecture Record

## 1. System Goal

CareerPilot is a personal career operating system that coordinates:

- job discovery
- job intelligence
- candidate data
- resume generation
- application preparation
- browser workflows
- human intervention
- application outcomes

The system should remain lightweight and inexpensive to operate.

---

# 2. High-Level Architecture

```text
                         JOB SOURCES
                             |
                             v
                  +----------------------+
                  | Job Ingestion        |
                  | Normalize/Dedupe     |
                  +----------+-----------+
                             |
                             v
                  +----------------------+
                  | Job Intelligence     |
                  | Gemini -> triage     |
                  | Claude -> deep eval  |
                  +----------+-----------+
                             |
                             v
                  +----------------------+
                  | Application Planner  |
                  +----------+-----------+
                             |
              +--------------+--------------+
              |                             |
              v                             v
      +---------------+             +---------------+
      | Resume Engine |             | Answer Engine |
      +-------+-------+             +-------+-------+
              |                             |
              +--------------+--------------+
                             |
                             v
                  +----------------------+
                  | Review / Safety Gate |
                  +----------+-----------+
                             |
                             v
                  +----------------------+
                  | Browser Workflow     |
                  | Playwright           |
                  +----------+-----------+
                             |
              +--------------+--------------+
              |                             |
              v                             v
          SUBMITTED                    HUMAN_REQUIRED
              |                             |
              +--------------+--------------+
                             |
                             v
                  +----------------------+
                  | Outcome Tracking      |
                  +----------------------+
## Architectural Goals

1. Lightweight cloud footprint.
2. Event-driven execution.
3. Strong deterministic guardrails.
4. Human recovery for uncertain browser states.
5. Versioned career artifacts.
6. Model-provider independence.
7. Full application auditability.

## Core Components

### Orchestrator

Cloudflare Worker:

- cron
- API endpoints
- job dispatch
- review callbacks
- rate limiting

### Database

Supabase PostgreSQL:

- candidate facts
- job state
- application state
- resume versions
- browser session metadata

### AI Layer

Provider abstraction:

```text
LLMProvider
├── GeminiProvider
└── ClaudeProvider
```

The rest of the system should not depend directly on a specific provider.

### Browser Layer

Playwright runner.

The browser layer owns:

- navigation
- form interaction
- screenshots
- browser context
- session persistence
- recovery

It must not own career decisions.

### Career Layer

Deterministic domain logic:

- resume rules
- application rules
- eligibility
- limits
- duplicate detection

---

## Trust Boundaries

```text
External job source
        |
        v
Untrusted job content
        |
        v
LLM extraction
        |
        v
Validated internal job model
        |
        v
Candidate/profile rules
        |
        v
Application plan
        |
        v
Browser
```

Job descriptions must never be allowed to override candidate safety rules or system instructions.

---

## State Ownership

Each subsystem owns its state:

```text
Career -> candidate facts
Resume -> resume versions
Job -> normalized job
Application -> application lifecycle
Browser -> browser session
Review -> human decisions
```

No component should infer state from another component's logs.

---

## Recovery Principle

Every long-running operation must have a persisted state.

Bad:

```text
browser automation
    |
    | crash
    v
start over
```

Good:

```text
browser automation
    |
    v
persist checkpoint
    |
    | crash
    v
restore checkpoint
    |
    v
continue
```

---

## Idempotency

Operations must be safe to retry.

Examples:

```text
create_job
create_application
save_resume_version
send_notification
resume_browser_session
```

Use idempotency keys where appropriate.

Never blindly retry a submission action.

---

## Security

Secrets:

- model API keys
- database credentials
- browser credentials
- notification credentials

must live in secret storage.

Never commit secrets.

Browser recovery tokens must be:

- cryptographically random
- hashed at rest
- short-lived
- revocable

---

## Resource Model

The default assumption is:

```text
No work -> no compute.
```

Use scheduled/event-driven invocations.

Browser sessions are created only for applications that passed the required checks.

---

## Model Routing

```text
Cheap / repetitive
        |
        v
Gemini

Complex / high-value
        |
        v
Claude

Deterministic
        |
        v
Go
```

LLMs propose decisions; deterministic code enforces policy.

