# CareerPilot — Resume Management

## Source of Truth

The candidate profile is the source of truth.

Resume files are generated artifacts.

```text
Candidate Profile
      |
      +--> Software Engineer Resume
      |
      +--> FDE Resume
      |
      +--> Job-specific variants
```

## Resume Types

### Software Engineer

Primary positioning:

- Go
- backend systems
- distributed systems
- GraphQL
- PostgreSQL
- Redis
- Kafka
- Angular
- production debugging

### Forward Deployed Engineer

Primary positioning:

- integrations
- operational requirements
- APIs/webhooks
- IoT/fleet systems
- ambiguity
- cross-service debugging
- domain modeling
- production problem solving

---

## Update Modes

### Manual

User explicitly requests an update.

### Scheduled Proposal

Cron identifies potentially relevant changes and creates a proposal.

### Automatic

Only allowed after the proposal workflow has demonstrated stable behavior and only for low-risk changes.

---

## Versioning

Every approved change creates a new version.

Never mutate an approved historical resume in place.

Example:

```text
software-engineer
v1
v2
v3
...
```

---

## Job-Specific Resume

A job-specific resume is derived from a base version:

```text
SWE v12
   +
Job XYZ
   |
   v
SWE v12-job-xyz
```

The generated variant must retain factual provenance.

---

## One-Page Requirement

Hard requirement:

```text
page_count == 1
```

If compilation produces more than one page:

1. remove low-value content
2. shorten redundant wording
3. reduce unnecessary spacing
4. retry compilation
5. if still >1 page, request review

Do not silently reduce typography below the established template standard.
