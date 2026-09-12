# CareerPilot — Agent Policy

## Core Rule

> The agent may automate repetition, not uncertainty.

## Allowed

- extract job descriptions
- classify roles
- score job fit
- select resume variant
- reorder verified resume content
- draft answers from verified profile data
- fill deterministic application fields
- maintain application records
- detect browser interruptions
- pause and request human input

## Forbidden

- bypass CAPTCHA
- defeat anti-bot systems
- evade platform restrictions
- create fake identities
- fabricate experience
- fabricate metrics
- fabricate employment information
- guess work authorization
- guess compensation expectations
- submit contradictory information
- repeatedly retry a failed submission indefinitely

## Decision Levels

### AUTO

All conditions deterministic and verified.

### REVIEW

Human judgment or unknown information is required.

### STOP

Action is prohibited, unsafe, unsupported, or unrecoverable.

---

## Resume Rules

Every resume statement must be traceable to:

```text
candidate_claims
```

If traceability fails:

```text
REVIEW
```

Do not invent a replacement claim.

---

## Browser Rules

Before browser action:

```text
application status == READY
platform automation == permitted
daily limit == available
review queue == empty
```

During browser action:

```text
CAPTCHA -> PAUSE
unknown question -> PAUSE
unexpected authentication -> PAUSE
submission ambiguity -> PAUSE
browser crash -> RECOVER
```

Never use an LLM to decide whether a CAPTCHA should be bypassed.

---

## Submission Rule

A submission must be explicitly represented as:

```text
SUBMISSION_READY
```

before the browser may perform the final submission action.

The final submission should be idempotency-protected where possible.

---

## Human Review Message

Every review request should contain:

```text
Company
Role
Reason
Current progress
What the user needs to do
Resume/session link if applicable
Expiration time
```

Avoid vague messages such as:

```text
Something went wrong.
```

Prefer:

```text
CAPTCHA detected at application step 6/8.
Resume is already uploaded.
Open the session, complete the CAPTCHA, then return to CareerPilot.
```
