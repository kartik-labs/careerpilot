# CareerPilot Data Model

## Principles
- PostgreSQL is the initial durable store.
- Business state is explicit.
- Important transitions are auditable.
- AI output is not automatically authoritative.
- Candidate facts require provenance.
- Application and browser state are separate.
- Operations should be idempotent.

## Candidate
### candidate_profiles
`id, version, status, created_at, updated_at`

### candidate_claims
`id, profile_id, category, claim_text, structured_value, source, provenance, confidence, verified, created_at`

Every resume claim must trace to candidate claims.

## Resume
### resume_variants
Examples: `SOFTWARE_ENGINEER`, `FORWARD_DEPLOYED_ENGINEER`

### resume_versions
`id, variant_id, profile_version, version_number, latex_source, pdf_artifact, page_count, validation_status, approval_status, change_summary, created_at, approved_at`

### resume_claims
Join table connecting resume versions to candidate claims.

## Jobs
### job_sources
`id, name, source_type, configuration_reference, active`

### jobs
`id, source_id, external_id, canonical_url, company, title, location, description, employment_type, remote_policy, compensation, posted_at, fingerprint, discovered_at`

### job_scores
`id, job_id, profile_version, score, recommendation, confidence, reasons, concerns, created_at`

### job_evaluations
`id, job_id, model_provider, model, prompt_version, result_reference, created_at`

## Applications
### applications
`id, job_id, candidate_profile_id, resume_version_id, state, platform, application_url, created_at, updated_at, submitted_at, outcome`

### application_answers
`id, application_id, question, answer, answer_type, confidence, source_claim_ids, review_required, approved`

### application_events
Append-only state-transition audit stream.

## Browser
### browser_sessions
`id, application_id, platform, state, current_url, started_at, last_checkpoint_at, expires_at, human_required, failure_reason`

### browser_checkpoints
`id, browser_session_id, state, url, screenshot_reference, checkpoint_data, created_at`

Never store raw credentials or cookies as ordinary business records.

## Notifications
### notifications
`id, recipient, type, payload, status, created_at, delivered_at`

## AI Audit
### model_runs
`id, provider, model, operation, prompt_version, input_reference, output_reference, status, latency_ms, token_usage, created_at`

## Global Audit
### audit_events
`id, actor, action, entity_type, entity_id, metadata, created_at`

## Integrity Rules
- Submitted application must reference an approved resume.
- Answers retain their source or explicit human approval.
- Resume cannot be approved when page validation fails.
- State transitions must be validated.
- Duplicate applications are rejected before submission.
- Audit events are append-only.
