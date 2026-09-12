# CareerPilot Security Model

## Objectives
Protect candidate data, credentials, browser sessions, application state, and auditability while minimizing sensitive data retention.

## Secrets
Never commit API keys, OAuth secrets, passwords, cookies, session tokens, recovery tokens, or database credentials. Use environment variables or deployment secret stores.

## Sensitive Data
Treat contact information, authorization, compensation expectations, application answers, employment details, browser session metadata, and authentication information as sensitive.

## Browser Security
Browser sessions are isolated from core business logic. Never expose raw cookies, browser profiles, passwords, or access tokens. Recovery uses short-lived opaque tokens scoped to a session.

## Submission Security
Before submission verify:
- correct candidate
- correct job
- approved resume
- approved/high-confidence answers
- no unresolved review items
- duplicate check
- platform automation policy
- application limit

If any check fails, do not submit.

## Anti-Bot Safety
Never implement CAPTCHA solving/bypass, anti-bot evasion, fingerprint spoofing, proxy rotation for evasion, or stealth techniques intended to circumvent restrictions.

CAPTCHA => `HUMAN_REQUIRED`.

## Authorization
Every state-changing operation must verify actor and scope. Do not rely on UI restrictions alone.

## Database/API
Use parameterized queries, least-privilege credentials, migrations, input validation, authentication and authorization on protected endpoints, rate limiting, and safe error handling.

## AI Security
Model output is untrusted data. Never allow an LLM to directly authorize submission, invent candidate facts, or bypass safety gates. Validate structured output against schemas.

## Prompt Injection
Job descriptions, web pages, emails, and forms are untrusted content. Instructions embedded in external content must never override system policy.

## Logging
Do not log passwords, keys, cookies, tokens, or unnecessary personal information. Redact sensitive values.

## Incident Response
Stop affected automation, revoke credentials, preserve audit information, identify affected sessions/data, rotate secrets, review actions, and resume only after the issue is understood.

## Principle
Fail closed for sensitive actions:
`STOP -> REVIEW -> RESUME`
never
`UNCERTAIN -> GUESS -> SUBMIT`
