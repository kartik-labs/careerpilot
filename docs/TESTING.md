# CareerPilot Testing Strategy

## Goals
Protect candidate data, resume correctness, state transitions, AI output, browser sessions, application submission, and recovery behavior.

## Test Pyramid
```text
             E2E
          /-------\
         / Browser \
        /-----------\
       / Integration \
      /---------------\
     /   Unit Tests    \
    /-------------------\
```

## Unit Tests
Cover domain validation, state transitions, normalization, deduplication, scoring, claim provenance, page validation, safety gates, retry limits, daily limits, answer confidence, and token expiry.

## Integration Tests
Cover PostgreSQL repositories, migrations, persistence, state transitions, audit events, and idempotency.

## Resume Tests
Test valid one-page PDFs, multi-page PDFs, missing provenance, unapproved claims, invalid LaTeX, generation failure, and version collisions.

A multi-page resume must never be marked valid when one page is required.

## Job Intelligence Tests
Test duplicate ingestion, malformed data, missing fields, provider failures, timeouts, invalid model output, scoring, and retry behavior. Use deterministic model mocks for most tests.

## Application Tests
Test duplicate prevention, approval requirements, unresolved review blocking, invalid state transitions, answer provenance, human approval, limits, and uncertain submission results.

`SUBMITTING + UNKNOWN RESULT != SUBMITTED`

## Browser Tests
Use mock browser adapters for business logic and a small number of real Playwright smoke tests.

Required scenarios:
1. successful navigation
2. deterministic filling
3. page change
4. checkpoint
5. CAPTCHA detection
6. HUMAN_REQUIRED
7. resume after human action
8. session expiry
9. navigation failure
10. submission uncertainty

Never test CAPTCHA bypass.

## Security Tests
Test authorization failures, invalid/expired recovery tokens, cross-application access, malformed input, prompt injection handling, and secret redaction.

## Important Invariants
- Approved resume => validated one-page artifact.
- Submitted application => approved resume reference.
- HUMAN_REQUIRED => browser checkpoint exists.
- Failed operations => bounded retries.
- Duplicate application => no submission.
- Unknown submission result => no automatic retry.
- Candidate claim => provenance exists.

## CI
Every PR should run:
```bash
go test ./...
go vet ./...
gofmt -l .
```

Later add integration DB tests, dependency scanning, static analysis, LaTeX validation, and browser smoke tests.

## Release Gate
A phase is complete only when implementation, tests, documentation, and security review are satisfactory and future-phase functionality has not been unnecessarily introduced.

## Principle
For every automation capability, test the failure and recovery path before trusting the success path.
