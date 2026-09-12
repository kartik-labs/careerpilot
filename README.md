# CareerPilot

CareerPilot is a personal AI career operating system: it discovers jobs, evaluates fit,
maintains truthful resume variants, tailors resumes to specific roles, prepares application
answers, and assists with browser-based applications — pausing for human input whenever
something is uncertain, unverifiable, or unsafe to automate.

It is not an autonomous job-spamming system. See [CLAUDE.md](CLAUDE.md) for the full set of
non-negotiable principles this project is built around.

## Documentation

- [CLAUDE.md](CLAUDE.md) — engineering and safety principles for anyone (human or agent) working on this repo
- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) — system architecture and trust boundaries
- [docs/IMPLEMENTATION-PLAN.md](docs/IMPLEMENTATION-PLAN.md) — phased build plan
- [docs/AGENT-POLICY.md](docs/AGENT-POLICY.md) — what automation is and isn't allowed to do
- [docs/RESUME-MANAGEMENT.md](docs/RESUME-MANAGEMENT.md) — resume versioning and generation rules
- [docs/BROWSER-RECOVERY.md](docs/BROWSER-RECOVERY.md) — browser session checkpointing and human handoff
- [docs/DECISIONS.md](docs/DECISIONS.md) — architecture decision records

## Repository layout

```text
cmd/            entry points (cmd/api is the HTTP server)
internal/       application code, not importable outside this module
  app/          dependency wiring
  config/       environment-based configuration
  logging/      structured logging
  httpserver/   HTTP handlers
  storage/      persistence abstractions (postgres/)
migrations/     SQL schema migrations
resumes/        resume templates and generated artifacts
prompts/        LLM prompt templates
workers/        on-demand background job code (browser runner, etc.)
tests/          cross-package integration tests
```

## Development

Requires Go 1.26+.

```bash
cp .env.example .env    # then fill in real values as needed
go build ./...
go vet ./...
go test ./...
gofmt -l .               # should print nothing
```

Run the API server locally:

```bash
go run ./cmd/api
curl http://localhost:8080/healthz
```

## Docker

Build and run the API server plus PostgreSQL with Docker Compose:

```bash
docker compose up -d --build
curl http://localhost:8080/healthz
docker compose down        # add -v to also drop the postgres volume
```

Set `GEMINI_API_KEY`/`ANTHROPIC_API_KEY` in your shell or a `.env` file before starting compose if
you need model provider calls to work; `docker-compose.yml` passes them through as empty by
default. Override `CAREERPILOT_HOST_PORT`/`CAREERPILOT_POSTGRES_HOST_PORT` if 8080/5432 are
already in use on your machine.

To build just the image (e.g. for pushing to a registry):

```bash
docker build -t careerpilot-api .
```

The image is a distroless, non-root, ~20MB static binary — no shell, no package manager. Only
`cmd/api` is containerized; the scheduler and browser-runner packages are library code with no
standalone binary yet (see docs/DECISIONS.md ADR-020).

## Phase discipline

CareerPilot is built incrementally, one phase at a time (see
[docs/IMPLEMENTATION-PLAN.md](docs/IMPLEMENTATION-PLAN.md)). Do not implement future-phase
functionality (job scraping, browser automation, autonomous submission) ahead of schedule.
