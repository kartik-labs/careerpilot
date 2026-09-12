# CareerPilot — Claude Code Instructions

## Project

CareerPilot is a personal AI career operating system.

Its purpose is to help a user:

1. Discover relevant jobs.
2. Evaluate job fit.
3. Maintain truthful resume variants.
4. Tailor resumes for specific opportunities.
5. Prepare application answers.
6. Assist with browser-based applications where permitted.
7. Pause for human intervention when uncertainty or verification occurs.
8. Track application outcomes.
9. Learn which jobs, resumes, and strategies produce better outcomes.

CareerPilot is NOT an autonomous job-spamming system.

The primary optimization target is:

qualified applications -> recruiter responses -> interviews -> offers

not:

applications submitted per day.

---

# Non-Negotiable Principles

## 1. Never fabricate candidate information

Never invent:

- experience
- employment history
- job titles
- responsibilities
- technologies
- metrics
- customers
- achievements
- certifications
- education
- salary information
- authorization information
- personal answers

Every resume claim must originate from the candidate profile or an explicitly approved source.

---

## 2. Human-in-the-loop

Use three decision classes:

AUTO
- deterministic
- low-risk
- reversible

REVIEW
- requires candidate confirmation

STOP
- unsafe
- unsupported
- ambiguous
- prohibited

Never convert REVIEW or STOP into AUTO merely to make the workflow continue.

---

## 3. CAPTCHA and anti-bot handling

CAPTCHAs are human verification.

Never implement:

- CAPTCHA solving
- CAPTCHA bypass
- proxy rotation for evasion
- fingerprint spoofing
- anti-bot circumvention
- infinite retries
- stealth techniques intended to evade platform controls

Correct behavior:

SAVE SESSION
-> PAUSE
-> NOTIFY USER
-> HUMAN COMPLETES VERIFICATION
-> RESUME SESSION

The browser session must be resumable.

---

## 4. Platform compliance

Before implementing browser automation for a platform, determine whether automation is permitted.

If automation is prohibited:

- do not automate submission
- allow preparation only
- provide a human handoff

---

# Architecture Principles

Prefer:

- event-driven workflows
- deterministic Go business logic
- PostgreSQL for durable state
- stateless/on-demand workers
- model providers behind interfaces
- explicit state machines
- idempotent operations
- auditability
- small deployable components

Avoid:

- unnecessary microservices
- permanent always-running agents
- hidden state
- provider-specific business logic
- uncontrolled model loops
- infinite retries

---

# AI Model Strategy

Use Gemini for:

- high-volume job triage
- inexpensive classification
- extraction
- preliminary scoring

Use Claude for:

- deep job analysis
- resume tailoring
- complex reasoning
- application-answer generation
- verification
- difficult decisions

Never allow an LLM to directly mutate critical application state without deterministic validation.

---

# Candidate Profile

The candidate profile is the source of truth.

Resume documents are generated views of that profile.

A resume version must retain:

- profile version
- creation timestamp
- target role
- changes
- source claims
- generated artifact
- validation result
- approval status

---

# Resume Rules

CareerPilot initially supports:

1. Software Engineer / Backend Engineer
2. Forward Deployed Engineer

Resume generation must preserve factual accuracy.

Never silently:

- invent metrics
- add technologies
- change dates
- change titles
- inflate responsibility
- claim ownership of work not supported by the profile

---

# Browser State Machine

Use explicit states:

QUEUED
STARTING
NAVIGATING
FILLING
VERIFYING
SUBMITTING
SUBMITTED

Exceptional states:

HUMAN_REQUIRED
PAUSED
FAILED
EXPIRED
CANCELLED

Never blindly retry SUBMITTING.

---

# Application Safety Gate

Before submission verify:

- candidate identity
- target job
- selected resume
- generated answers
- unresolved REVIEW items
- duplicate application status
- platform automation policy
- daily application limit

Only then may submission proceed.

---

# Initial Resource Limits

daily:

jobs_discovered: 100
deep_evaluations: 30
applications: 15
browser_sessions: 10

per_application:

max_retries: 2
max_session_minutes: 30

These must be configurable.

---

# Engineering Standards

Use Go for backend/core business logic.

Prefer:

- small packages
- dependency injection
- interfaces around external systems
- context.Context
- structured logging
- explicit errors
- table-driven tests
- integration tests for persistence
- idempotent commands
- migrations for schema changes

Do not add dependencies without justification.

---

# Development Workflow

Before implementing a feature:

1. Read relevant documentation.
2. Inspect existing repository structure.
3. Identify existing abstractions.
4. Propose the smallest coherent implementation.
5. Implement.
6. Add tests.
7. Run formatting.
8. Run tests.
9. Run static analysis where applicable.
10. Update documentation.
11. Summarize changes.

Never rewrite existing working code merely for stylistic preference.

---

# Phase Discipline

CareerPilot is implemented incrementally.

Do not implement future phases prematurely.

If working on Phase 1:

- do not build browser automation
- do not build job scraping
- do not build autonomous application submission

Keep the repository shippable after every phase.

---

# Decision Logging

Important architecture decisions must be recorded in:

docs/DECISIONS.md

Each decision should contain:

- context
- decision
- alternatives
- rationale
- consequences

---

# When uncertain

STOP and explain the uncertainty.

Do not make up requirements.

Do not assume a platform permits automation.

Do not fabricate candidate information.

Do not silently change architecture.