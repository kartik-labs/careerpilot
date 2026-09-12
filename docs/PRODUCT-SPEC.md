# CareerPilot Product Specification

## Product Vision
CareerPilot is a personal AI career operating system for discovering relevant jobs, evaluating fit, managing truthful resume variants, preparing applications, assisting with permitted browser workflows, and learning from outcomes.

Primary optimization target:
`qualified applications -> recruiter responses -> interviews -> offers`

## Core Capabilities
- Job intelligence: ingest, normalize, deduplicate, score, and explain jobs.
- Resume management: source-of-truth candidate profile, resume variants, tailoring, versioning, LaTeX compilation, one-page validation, approval.
- Application preparation: resume selection, grounded answers, confidence classification, review queue, duplicate prevention.
- Browser assistance: permitted workflows, persistent sessions, checkpoints, human handoff, resumability.
- Analytics: funnel, outcomes, resume performance, job-score calibration.

## Human Review
Review is mandatory for ambiguous questions, authorization/sponsorship, relocation, compensation, legal declarations, conflicting data, CAPTCHA/human verification, and unknown platform policy.

## Non-Goals
- CAPTCHA solving or bypass
- anti-bot evasion
- fabricated candidate claims
- mass application spamming
- blind retries after uncertain submission
- unnecessary distributed infrastructure
- automatic modification of factual candidate data from outcomes

## Application States
`PREPARING -> REVIEW -> READY -> STARTING -> NAVIGATING -> FILLING -> VERIFYING -> SUBMITTING -> SUBMITTED`

Exceptional states:
`HUMAN_REQUIRED`, `PAUSED`, `FAILED`, `EXPIRED`, `CANCELLED`

## Initial Limits
```yaml
daily:
  jobs_discovered: 100
  deep_evaluations: 30
  applications: 15
  browser_sessions: 10
per_application:
  max_retries: 2
  max_session_minutes: 30
```

## MVP Acceptance Criteria
1. Maintain a verified profile.
2. Generate approved resume variants.
3. Ingest and evaluate jobs.
4. Prepare application packages.
5. Review uncertain information.
6. Run permitted browser workflows.
7. Recover from human verification.
8. Record outcomes.
9. Inspect an audit trail.

## Product Principle
Automate repetitive career work, but never automate uncertainty.
