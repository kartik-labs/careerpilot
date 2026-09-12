# CareerPilot — Architecture Decisions

## ADR-001: Go module path

**Context**: The repository needed a Go module path to begin any Go development.

**Decision**: Use `github.com/kartik-labs/careerpilot`, matching the repository's git remote
(`https://github.com/kartik-labs/careerpilot.git`).

**Alternatives**: A placeholder/local module path (e.g. `careerpilot`) was considered but
rejected — it would need to be renamed once published, breaking all import paths.

**Consequences**: All internal packages import under this path. Renaming the GitHub org/repo
later would require a module path migration.

---

## ADR-002: PostgreSQL driver — pgx (stdlib mode)

**Context**: `docs/ARCHITECTURE.md` and `docs/IMPLEMENTATION-PLAN.md` specify PostgreSQL/Supabase
as the durable store but do not name a driver.

**Decision**: Use `github.com/jackc/pgx/v5/stdlib`, registered as a `database/sql` driver, rather
than pgx's native (non-stdlib) API.

**Alternatives**:
- `lib/pq` — unmaintained, rejected.
- pgx native API (`pgxpool.Pool`) — more performant but couples repository code directly to pgx
  types instead of the standard library `database/sql` interfaces, working against
  CLAUDE.md's "interfaces around external systems" principle at this stage.

**Consequences**: Repository code depends only on `*sql.DB`/`*sql.Tx`. If pgx-specific features
(e.g. native COPY, LISTEN/NOTIFY) become necessary later, this can be revisited.

---

## ADR-003: HTTP routing — standard library only

**Context**: Phase 0 needs a single `/healthz` endpoint; future phases will add more HTTP
surface area for review callbacks and application APIs.

**Decision**: Use `net/http`'s `http.ServeMux` (Go 1.22+ method-pattern routing) instead of a
third-party router.

**Alternatives**: chi, gorilla/mux — both add a dependency for functionality the standard
library now covers for CareerPilot's expected route complexity.

**Consequences**: If route matching needs (e.g. regex path segments, middleware chaining
ergonomics) grow significantly, a router library can be introduced without restructuring
handlers, since `Server.Handler()` already returns a plain `http.Handler`.

---

## ADR-004: Configuration via environment variables only

**Context**: CareerPilot targets Cloudflare Workers + on-demand runners (docs/ARCHITECTURE.md
§ Resource Model), not long-lived servers with config files.

**Decision**: `internal/config` reads exclusively from environment variables, with defaults and
an `.env.example` for local development. No config file format (YAML/TOML) is supported.

**Alternatives**: A structured config file was considered but rejected as unnecessary complexity
for the current deployment target and phase.

**Consequences**: Local development requires exporting env vars or using a `.env` loader
(e.g. `direnv`, `docker compose --env-file`) — none is bundled into the Go binary itself to avoid
an extra dependency; developers copy `.env.example` and source it via their tool of choice.

---

## ADR-005: Dependency injection via a plain `internal/app` wiring package

**Context**: CLAUDE.md requires "dependency injection" and "interfaces around external systems"
without mandating a specific pattern or DI framework.

**Decision**: Use a single `internal/app.New(ctx, cfg)` constructor function that wires concrete
implementations (logger, Postgres connection, HTTP server) and returns an `*App` struct. No DI
framework (wire, fx, dig) is introduced.

**Alternatives**: A DI framework was considered but rejected per CLAUDE.md's "do not add
dependencies without justification" and "prefer the simplest architecture" — a single wiring
function is sufficient at this scale and avoids reflection-based magic.

**Consequences**: As the dependency graph grows, `app.New` will grow with it. Revisit if wiring
logic becomes unwieldy (e.g. >~15 dependencies or conditional wiring graphs).

---

## ADR-006: LaTeX templating with `<< >>` delimiters

**Context**: Go's `text/template` default delimiters are `{{ }}`, which collide constantly with
LaTeX's own brace syntax (`\textbf{...}`), making templates unreadable and error-prone.

**Decision**: `internal/resume/latex.Render` configures `text/template` with `<<` `>>` delimiters.

**Alternatives**: A dedicated LaTeX templating library was considered but rejected — no
widely-used, well-maintained Go option exists that improves meaningfully on stdlib
`text/template` with custom delimiters.

**Consequences**: All `.tex.tmpl` files under `resumes/templates/` must use `<<`/`>>` instead of
`{{`/`}}`. This is enforced by convention and by `TestBaseTemplate_RendersValidLatex`.

---

## ADR-007: PDF page counting via `pdfinfo`, not a byte-scan heuristic

**Context**: The one-page requirement (docs/RESUME-MANAGEMENT.md) is a hard gate — an incorrect
page count could let a two-page resume pass as one page. A naive scan for `/Type /Page` markers
in the raw PDF bytes is unreliable against compressed object streams, which `pdflatex` output
commonly uses.

**Decision**: Shell out to `pdfinfo` (poppler-utils) to read the authoritative `Pages:` count.

**Alternatives**: A pure-Go PDF parsing library was considered but rejected for now — it's a
non-trivial dependency to vet for correctness against compressed streams, and `pdfinfo` is a
well-established, already-battle-tested tool typically available alongside any TeX Live
installation.

**Consequences**: The on-demand LaTeX compilation environment (CI, and later the resume-update
worker) must install `poppler-utils` alongside the TeX Live toolchain. `internal/resume/latex`
tests skip integration coverage of this path when `pdfinfo`/`latexmk` aren't installed locally,
relying on CI to exercise the real toolchain.

---

## ADR-008: Resume templates are shared across SWE/FDE variants

**Context**: docs/RESUME-MANAGEMENT.md defines two resume types (Software Engineer, Forward
Deployed Engineer) with different *content positioning*, not different page layouts.

**Decision**: Ship a single LaTeX template (`resumes/templates/base.tex.tmpl`) parameterized by
`latex.Data`. Variant selection (`resume.VariantType`) determines which candidate claims and
phrasing get fed into that template, not which template file is used.

**Alternatives**: Separate `.tex.tmpl` files per variant were considered but rejected as
premature duplication — nothing in the current spec requires a different visual layout per
variant. If SWE/FDE resumes need genuinely different layouts later, split the template then.

**Consequences**: If a variant ever needs a structurally different layout (not just different
content), a second template file plus a lookup by `VariantType` can be added without touching
the pipeline's public API (`GenerateInput.TemplateName`/`TemplateSource` are already
caller-supplied).

---

## ADR-009: Gemini/Claude providers are mocks pending real API credentials

**Context**: Phase 2 (Job Intelligence) requires routing triage through Gemini and deep
evaluation through Claude (CLAUDE.md "AI Model Strategy"). No API keys or vendor SDKs are
available in this environment/repo.

**Decision**: `internal/modelprovider.GeminiProvider` and `ClaudeProvider` implement the
`Provider` interface via an injectable `GenerateFunc`, defaulting to `ErrNotConfigured` when
unset. No HTTP calls to any vendor endpoint are made.

**Alternatives**: Inventing a plausible-looking HTTP client against Gemini/Claude APIs from
memory was considered and rejected — CLAUDE.md and the phase instructions explicitly prohibit
guessing at unavailable external integrations; a wrong contract would fail silently or
confusingly once real credentials are added later.

**Consequences**: `internal/job.Triager`/`Evaluator` and the `Pipeline` are fully testable today
via `GenerateFunc`. Wiring a real Gemini/Claude HTTP client is a drop-in replacement — implement
`Provider` against the real SDK/API and pass it into the existing `Triager`/`Evaluator`
constructors; no business logic changes.

---

## ADR-010: No SQL migrations yet for job/evaluation/application persistence

**Context**: Phase 2 and Phase 3 both define storage-shaped interfaces (`job.Store`,
`job.EvaluationStore`, `modelprovider.RunStore`, `application.Store`) but Postgres schema
migrations for these tables were not explicitly requested by either phase prompt, and
CLAUDE.md's "Phase Discipline" warns against implementing future phases prematurely.

**Decision**: Keep these as Go interfaces with in-memory fakes for tests only. Concrete
Postgres-backed implementations and their migrations are deferred to a dedicated persistence
phase.

**Alternatives**: Writing migrations now (mirroring docs/IMPLEMENTATION-PLAN.md section 10/11's
table lists) was considered, since `internal/storage/postgres` already exists from Phase 0.
Rejected for now to avoid guessing at schema details (indexes, constraints, JSON vs. relational
shape for `Score`/`RawData`) that should be driven by an explicit persistence-focused prompt.

**Consequences**: `cmd/api` currently has no working Postgres-backed job/application
functionality end-to-end — only the domain logic and interfaces exist. Wiring real persistence
requires: SQL migrations under `migrations/`, `postgres`-backed implementations of each `Store`
interface, and updating `internal/app` wiring.

---

## ADR-011: Application state machine models the full lifecycle up front

**Context**: Phase 3's minimum required states (docs pasted into the phase prompt) run from
DISCOVERED through CANCELLED, including browser-execution states (STARTING..SUBMITTED,
HUMAN_REQUIRED) that belong to a not-yet-built browser runner.

**Decision**: `internal/application.Status`/`validTransitions` model the complete state graph
now, including the post-READY browser states, even though Phase 3 does not implement anything
that drives transitions past READY.

**Alternatives**: Modeling only DISCOVERED..READY and adding the rest when the browser runner
phase begins was considered. Rejected because the phase prompt explicitly listed the full state
set as a minimum, and a partial state machine would need a breaking change (not just an
addition) once the browser runner needs to plug into it — enums split across two phases risk
diverging from CLAUDE.md's "Browser State Machine" naming.

**Consequences**: `HUMAN_REQUIRED`'s resume transitions (back to NAVIGATING/FILLING/VERIFYING)
are a simplification — the model does not yet track *which* in-flight step a paused application
should resume into. The future browser-recovery implementation (docs/BROWSER-RECOVERY.md) will
need a `checkpoint`/`current_step` field on `Application` or a separate `BrowserSession` type to
resume correctly; `Transition` itself does not need to change.

---

## ADR-012: Sensitive answer categories are hardcoded to always-REVIEW

**Context**: docs/AGENT-POLICY.md and CLAUDE.md forbid guessing work authorization, sponsorship,
relocation, compensation, and similar answers, regardless of how confident an evaluator might be.

**Decision**: `internal/application.Classify` checks category membership in a fixed
`alwaysReview` set *before* looking at confidence at all — a "high confidence" work-authorization
answer is still forced to REVIEW.

**Alternatives**: Letting confidence alone gate AUTO/REVIEW (i.e., trusting a sufficiently
confident model output even for sensitive categories) was considered and rejected outright — it
directly contradicts CLAUDE.md's non-negotiable principle against guessing these categories,
and confidence is a model-reported signal, not a guarantee.

**Consequences**: Even if a future evaluator becomes very good at inferring, say, relocation
willingness from profile data, that category will still require human confirmation on every
application unless this list is deliberately revisited as an explicit policy change (recorded as
a new ADR, not a silent code change).

---

## ADR-013: Playwright via `playwright-community/playwright-go`, isolated behind `browser.Driver`

**Context**: Phase 4 requires real browser automation. No browser binaries or display are
available in this development/build environment, and CLAUDE.md/docs/ARCHITECTURE.md require the
browser layer to be isolated from business logic.

**Decision**: `internal/browser.Driver` is the sole interface business logic (and
`internal/browser.Runner`) depends on. `PlaywrightDriver` implements it using
`github.com/playwright-community/playwright-go` (pinned to v0.5200.0, the latest version whose
module path matches its current import path — later tags were published under the renamed
`github.com/mxschmitt/playwright-go` path and don't resolve under the community path). `FakeDriver`
implements the same interface as a fully scriptable in-memory double for tests.

**Alternatives**: chromedp was considered — lighter weight, CDP-native — but Playwright's
multi-engine support and first-class Go bindings better match "Use Playwright" as stated in the
phase prompt.

**Consequences**: Running `PlaywrightDriver` for real requires `playwright.Install()` (downloads
browser binaries) as a separate operational step — not invoked automatically by this code, since
that's an infrastructure/deployment concern, not application startup behavior. All Phase 4 tests
run against `FakeDriver` only; `PlaywrightDriver` has no automated test coverage in this
environment and should be exercised manually or in a CI image with browsers installed before
relying on it in production.

---

## ADR-014: Runner never re-navigates on Resume

**Context**: The initial `Runner.Resume` implementation called the same `navigateAndFill` helper
used by `Start`, which re-issued `Driver.Navigate` to the session's `CurrentURL`. A test
(`TestRunner_Resume`) caught that this could re-trigger the exact CAPTCHA/blocking state that
paused the session in the first place, and — worse — a real re-navigation could discard
in-progress form state on some platforms.

**Decision**: `Resume` only re-detects state, then calls a new `fillAndCheckpoint` helper (fill +
detect + checkpoint) shared with `Start`'s tail — no `Navigate` call after the initial one.

**Alternatives**: Making `FakeDriver.NavigateStates` cycle indefinitely instead of repeating the
last entry was considered, but that would have hidden the actual bug (an unwanted second
navigation) rather than fixing it.

**Consequences**: If a platform's application flow genuinely requires re-navigating after a
human resolves a CAPTCHA (uncommon, but possible for some ATSs), that must be a deliberate,
separate code path — not the default `Resume` behavior.

---

## ADR-015: Scheduled tasks are plain functions, not a self-looping scheduler

**Context**: Phase 5 requires scheduled job discovery/evaluation and resume proposals, but
explicitly forbids an infinite loop, and docs/ARCHITECTURE.md places scheduling in a Cloudflare
Worker's cron trigger, not a long-running Go process.

**Decision**: `internal/scheduler.DiscoveryTask`, `EvaluationTask`, and `ResumeProposalTask` each
expose a single `Run(ctx, ...)` method that does exactly one pass and returns. Nothing in this
package starts a goroutine, timer, or loop — the caller (a cron-triggered handler, a CLI command,
a test) decides when `Run` is invoked.

**Alternatives**: A `Scheduler` type with a `Start()`/`Stop()` and an internal ticker was
considered, matching a typical long-running-service pattern. Rejected because it contradicts both
the phase's explicit "never allow an infinite loop" instruction and CLAUDE.md's "stateless/on-demand
workers" / "No work -> no compute" principles — CareerPilot's target deployment has no
process that is expected to stay up between invocations.

**Consequences**: Wiring real cron requires an external trigger (Cloudflare Worker cron, a
`cmd/scheduler` CLI invoked by system cron, etc.) that calls these `Run` methods — not implemented
in this phase since no deployment target was specified.

---

## ADR-016: Notification delivery is a log-backed stub pending a real channel

**Context**: Phase 5 requires application-review and human-required notifications, but no
concrete channel (email/SMS/Slack/push) or credentials were provided.

**Decision**: `internal/notification.Service` always persists a `Notification` record via `Store`
and then calls an injected `Sender`. The only `Sender` implementation shipped is `LogSender`,
which writes a structured log line. `*Service` also satisfies `internal/browser.Notifier`
directly, so it can be passed straight into a `browser.Runner`.

**Alternatives**: Guessing at a specific provider's API (e.g. SendGrid, Twilio) was rejected for
the same reason as Phase 2's model providers — no credentials or contract were given, and
CLAUDE.md instructs against inventing external integrations.

**Consequences**: Notifications currently only appear in logs. Adding a real channel means
writing one more `Sender` implementation and wiring it into `internal/app` — no changes needed to
`Service`, `browser.Runner`, or any caller.

---

## ADR-017: Daily limits reuse the Phase 0 `config.Limits` struct

**Context**: Phase 5 re-specifies the same daily limits (jobs_discovered: 100, deep_evaluations:
30, applications: 15, browser_sessions: 10) and per-application limits (max_retries: 2,
max_session_minutes: 30) that Phase 0's `internal/config.Limits` already defined and made
configurable via environment variables.

**Decision**: No new limits configuration was added. `job.Pipeline.Limits`,
`browser.Runner.Limits`, and `scheduler.RetryPolicy` all take plain struct values that callers
populate from the existing `config.Config.Limits` — one source of configuration, not a
parallel one.

**Consequences**: Wiring `cmd/api`/a future scheduler entry point must pass
`cfg.Limits.JobsDiscoveredPerDay` etc. into each component's constructor; there is no separate
Phase 5 config surface to keep in sync.

---

## ADR-018: Outcome tracking is append-only stage history, not a mutable status field

**Context**: Phase 6 must track funnel progression (discovered → qualified → ... → offer/
rejected/withdrawn) per application, and compute analytics like funnel counts and rates over
that history.

**Decision**: `outcome.Record.Stages` is an append-only `[]StageEvent`; `AppendStage` always adds
a new entry and never removes or overwrites prior ones. `FinalOutcome` is derived (set only when
a terminal stage is appended), not independently settable.

**Alternatives**: A single mutable `Record.Status` field (like `job.Status`/`application.Status`)
was considered, mirroring those packages' state machines. Rejected for outcome tracking
specifically: analytics need to know *when* each stage was reached (e.g. days-to-response), and a
single overwritten status field would discard that timing data that funnel/rate analytics depend
on.

**Consequences**: Any query needing "current stage" must take the last entry in `Stages`
(`FinalOutcome` covers the common terminal case); there is no O(1) "status" field to filter on
without at least a small helper, which is what `HasStage` and `Funnel` provide.

---

## ADR-019: Analytics are pure functions over `[]Record`, with no write path to profile/resume

**Context**: Phase 6 explicitly requires: "Do not automatically change resume facts based on
outcomes... Changes to the candidate profile or resume require review."

**Decision**: Every function in `internal/outcome/analytics.go` and `recommendation.go` takes
`[]Record` (or similar read-only input) and returns a value — none of them accept or call into
`internal/candidate.Store` or `internal/resume.Store`. `Recommendation` is a plain data struct
describing a finding and a suggestion string; nothing consumes it automatically.

**Alternatives**: Wiring `RecommendLowPerformingResumeVersions` to automatically create a
`scheduler.ResumeProposal` was considered, since Phase 5 already has that type. Rejected —
auto-creating even a *proposal* from outcome data blurs the line Phase 6 draws ("may recommend,"
not "may propose changes into the existing review queue unprompted"); a human explicitly invoking
that connection is a deliberate integration step for a future phase, not implied by "recommend."

**Consequences**: Turning an `outcome.Recommendation` into an actual resume change requires a
human (or an explicitly human-triggered action) to read it and separately kick off the existing
`resume` package's DRAFT→...→APPROVED flow — there is no automatic path from analytics output to
a mutated candidate fact.

---

## ADR-020: Docker covers only `cmd/api`; no browser/Chromium in the image

**Context**: The repo has one runnable binary (`cmd/api`); the scheduler, browser runner, and
notification packages are library code invoked by tests, not wired into any `cmd/` entrypoint.
Bundling a real browser (Chromium, for `browser.PlaywrightDriver`) into the API image would add
several hundred MB and a slower build for functionality nothing currently calls at runtime.

**Decision**: `Dockerfile` is a two-stage build — `golang:1.26` compiles a static
(`CGO_ENABLED=0`) binary, then `gcr.io/distroless/static-debian12:nonroot` runs it. Final image
is ~20MB, no shell, runs as a non-root user. `docker-compose.yml` adds a `postgres:16-alpine`
service wired to the same `CAREERPILOT_DATABASE_URL` env var `internal/config` already reads.

**Alternatives**: An `alpine`-based runtime image was considered for shell-based debugging
access; rejected in favor of distroless's smaller attack surface, since CGO is disabled anyway
(pgx's stdlib driver needs no C library) and nothing in `cmd/api` needs a shell at runtime.
Bundling Playwright/Chromium into this same image was also considered and explicitly deferred —
when a browser-runner entrypoint exists, it should get its own Dockerfile/image (larger, different
base) rather than bloating the API image every deploy needs to pull.

**Consequences**: Verified end-to-end: `docker compose up -d --build` starts Postgres, waits for
its healthcheck, starts the API, and `GET /healthz` returns `{"status":"ok"}` against the real
containerized Postgres connection. Adding a scheduler or browser-runner binary later means a new
`cmd/` entrypoint plus (for the browser runner specifically) a separate Dockerfile that installs
Playwright's browser binaries — not an extension of this one.
