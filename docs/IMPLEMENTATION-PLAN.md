# CareerPilot — Implementation Plan

## 0. Objective

Build a lightweight personal AI career automation system that manages resumes, evaluates job postings, prepares applications, and performs permitted browser workflows with robust human recovery.

Primary constraint:

> The system must fail safely and recover cleanly rather than becoming stuck.

---

# 1. Phase 1 — Foundation

## 1.1 Candidate Profile

Create a structured candidate profile containing:

- identity/contact information
- education
- employment history
- verified technical skills
- projects
- certifications
- target roles
- geographic preferences
- compensation preferences
- work authorization answers
- reusable application answers

Add a `verified_claims` concept.

Each claim should have:

```text
claim
source
confidence
allowed_for_resume
notes
```

Add a `forbidden_claims` list to prevent hallucinated resume content.

---

## 1.2 Resume Model

Represent a resume as:

```text
resume
├── type
├── version
├── source_tex
├── generated_pdf
├── profile_version
├── created_at
├── approved_at
├── page_count
├── changes
└── status
```

Types:

```text
software-engineer
fde
```

---

## 1.3 Resume Update Engine

Support:

### Manual update

```text
POST /resumes/update
```

Input:

```json
{
  "resume_type": "software-engineer",
  "change_request": "Add recent Traccar and reporting work"
}
```

### Scheduled update

Cron triggers a proposal generation job.

The agent must produce:

```text
proposed_changes
reason
affected_sections
page_count_estimate
risk
```

No automatic approval during MVP.

---

## 1.4 LaTeX Pipeline

Pipeline:

```text
.tex
 |
 v
latexmk / pdflatex
 |
 v
PDF
 |
 v
page count
 |
 +--> 1 page -> PASS
 |
 +--> >1 page -> compression attempt
                         |
                         +--> 1 page -> PASS
                         |
                         +--> >1 page -> REVIEW
```

Never solve overflow by arbitrarily shrinking the font.

---

# 2. Phase 2 — Job Intelligence

## 2.1 Job Schema

```text
jobs
├── id
├── source
├── source_job_id
├── url
├── company
├── title
├── location
├── description
├── discovered_at
├── normalized_hash
├── fit_score
├── recommendation
└── status
```

---

## 2.2 Triage Agent

Gemini handles inexpensive first-pass extraction.

Output:

```json
{
  "role_type": "backend",
  "fit_score": 87,
  "hard_fail": false,
  "matched_skills": [],
  "missing_skills": [],
  "risk_flags": [],
  "recommended_resume": "software-engineer",
  "recommendation": "apply"
}
```

Hard filters run before expensive Claude calls.

---

## 2.3 Deep Evaluation Agent

Claude evaluates promising jobs.

It should examine:

- actual responsibilities
- required experience
- technical match
- domain match
- transferable experience
- role seniority
- location
- application risk
- likely resume positioning

Return:

```text
APPLY
REVIEW
REJECT
```

with reasons.

---

# 3. Phase 3 — Resume Tailoring

## Rules

The tailoring agent can:

- reorder bullets
- choose relevant bullets
- shorten bullets
- emphasize matching technologies
- select SWE/FDE variant

It cannot:

- invent metrics
- invent responsibilities
- change title
- change employment dates
- invent customer-facing experience
- add unsupported technologies

Every generated claim must map to the verified candidate profile.

---

# 4. Phase 4 — Application Preparation

## Application Profile

Store reusable answers.

Example categories:

```text
personal
education
employment
work_authorization
relocation
compensation
experience
preferences
```

Each answer has:

```text
question_pattern
answer
confidence
requires_review
```

---

## Answer policy

Automatically answer only when:

```text
known == true
AND
confidence == high
AND
question is deterministic
```

Otherwise:

```text
REVIEW
```

---

# 5. Phase 5 — Browser Runner

Use Playwright.

## State Machine

```text
QUEUED
  |
  v
STARTING
  |
  v
NAVIGATING
  |
  v
FILLING
  |
  +---- CAPTCHA ----> HUMAN_REQUIRED
  |
  +---- UNKNOWN ----> HUMAN_REQUIRED
  |
  v
VERIFYING
  |
  v
SUBMITTING
  |
  v
SUBMITTED
```

Failure states:

```text
PAUSED
FAILED
EXPIRED
CANCELLED
```

---

# 6. CAPTCHA Recovery

## Detection

Detect explicit CAPTCHA/verification states without attempting to solve them.

When detected:

1. Save browser state.
2. Save current URL.
3. Save current application step.
4. Save a screenshot.
5. Persist session metadata.
6. Mark application `HUMAN_REQUIRED`.
7. Notify user.
8. Stop browser activity.

Notification:

```text
CAPTCHA detected

Company: X
Role: Y
Progress: 82%

Resume application:
<temporary session URL>

Session expires:
<time>
```

---

## Resume

After the user completes the CAPTCHA:

```text
RESUME_SESSION(session_id)
```

The runner restores the browser context and continues from the recorded state.

Never restart the application unless recovery is impossible.

---

# 7. Session Security

Session URLs must be:

- random
- short-lived
- single-purpose
- revocable
- HTTPS-only

Never expose browser cookies or raw authentication tokens to the user interface.

Prefer a capability token:

```text
/recovery/<random-token>
```

The server maps the token to the internal session.

---

# 8. Application Guardrails

Before submission:

```text
Candidate identity verified
Resume verified
Application answers verified
Duplicate check passed
No unresolved review item
Platform automation policy allows action
Daily limit not exceeded
```

Only then:

```text
SUBMIT
```

---

# 9. Daily Resume Automation

Recommended initial schedule:

```text
08:00 Asia/Kolkata
```

Process:

```text
Read recent career activity
        |
        v
Identify resume-relevant changes
        |
        v
Generate proposal
        |
        v
Verify claims
        |
        v
Compile
        |
        v
Check one page
        |
        v
Notify user
```

Initial mode:

```text
PROPOSE
```

Later optional mode:

```text
AUTO_APPROVE_LOW_RISK
```

---

# 10. Storage

Minimum tables:

```text
candidate_profiles
candidate_claims
resume_versions
resume_changes
jobs
job_evaluations
applications
application_answers
browser_sessions
review_requests
notifications
career_events
```

Add indexes for:

```text
jobs.normalized_hash
applications.job_id
applications.company
browser_sessions.token_hash
review_requests.status
```

---

# 11. Lightweight Cloud Design

Avoid long-running services wherever possible.

Use:

```text
Cloudflare Worker
    |
    +-- cron
    +-- API
    +-- orchestration
    |
    v
Supabase
    |
    +-- PostgreSQL
    +-- storage
```

The browser runner should be on-demand.

No always-on browser.

No always-on model agents.

No infinite workers.

---

# 12. Model Strategy

## Gemini

Use for:

- job extraction
- initial classification
- cheap filtering
- bulk processing

## Claude

Use for:

- deep job evaluation
- resume tailoring
- complex application answers
- final verification

## Go

Use for:

- deterministic business logic
- state machine
- validation
- storage
- orchestration
- safety rules

Do not delegate deterministic rules to an LLM.

---

# 13. Testing Strategy

## Unit tests

Test:

- job normalization
- duplicate detection
- fit scoring
- resume claim validation
- page-count validation
- state transitions
- daily limits
- session token validation

## Integration tests

Test:

```text
job -> triage -> evaluation
profile -> resume -> compile
application -> browser state
CAPTCHA -> pause -> resume
```

## Failure tests

Explicitly simulate:

- browser crash
- network timeout
- CAPTCHA
- unknown question
- expired session
- duplicate application
- model timeout
- model hallucinated claim
- PDF compilation failure
- second-page resume

---

# 14. Observability

Every agent action should have:

```text
run_id
agent
input_reference
output
decision
model
latency
token usage
cost estimate
timestamp
```

Never store secrets in logs.

---

# 15. Rollout Strategy

### Stage A

Resume management only.

### Stage B

Job analysis, no applications.

### Stage C

Application preparation, human submission.

### Stage D

Browser automation on permitted application systems.

### Stage E

Controlled auto-submit.

### Stage F

Outcome learning.

Do not skip directly to Stage E.

---

# 16. Definition of Done for MVP

MVP is complete when:

- [ ] Candidate profile exists.
- [ ] SWE resume exists as a versioned artifact.
- [ ] FDE resume exists as a versioned artifact.
- [ ] Manual resume updates work.
- [ ] Scheduled resume proposals work.
- [ ] LaTeX compiles automatically.
- [ ] Page count is validated.
- [ ] Job can be scored.
- [ ] Resume can be selected.
- [ ] Application can be prepared.
- [ ] Unknown answers trigger review.
- [ ] Browser sessions persist.
- [ ] CAPTCHA causes a safe pause.
- [ ] User can resume a paused session.
- [ ] Duplicate applications are blocked.
- [ ] Daily limits are enforced.
- [ ] All application actions are auditable.

---

# 17. Future Extensions

After the MVP:

- recruiter response tracking
- interview tracking
- resume A/B analysis
- company prioritization
- salary intelligence
- interview preparation
- personalized application strategy
- career trend analysis
- outcome-based resume optimization
