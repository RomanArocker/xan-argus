# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

XAN-Pythia is a multi-tenant asset/user/license/service management system for an IT services business. MVP scope: CRUD, search, and overview — no authentication.

Design spec: `@docs/superpowers/specs/2026-03-27-xan-pythia-design.md`
Data model: `@docs/data-model.mmd`

## Project Learnings

Read `.claude/learnings.md` at session start — contains project-specific insights, fixed bugs, and known gotchas.

## Tech Stack

- **Backend:** Go (stdlib + pgx for database)
- **Database:** PostgreSQL 18.3
- **Frontend:** HTMX + Go HTML templates
- **Migrations:** goose (pure SQL, run on startup)
- **Deployment:** Docker Compose (multi-stage build)

## Project Structure

```
cmd/server/main.go          # Entry point
internal/handler/            # HTTP parsing, routing, JSON/template rendering
internal/repository/         # SQL via pgx, returns model structs
internal/model/              # Plain Go structs, no methods
db/migrations/               # goose SQL migrations
web/templates/               # Go HTML templates
web/static/                  # CSS/JS assets
```

No service layer in MVP — handlers call repositories directly.

## Language

- **Business language: English** — all code, comments, commit messages, UI labels, and documentation must be in English.

## Architecture Principles

- **PostgreSQL-first:** leverage DB constraints, triggers, JSONB, GIN indexes directly. Keep Go layer thin.
- **Minimal dependencies:** prefer stdlib over third-party packages wherever reasonable.
- **Strict error handling:** always wrap errors with `fmt.Errorf("context: %w", err)`, never ignore errors silently.
- **API-first:** REST JSON under `/api/v1/`, standard pagination with `?limit=&offset=`, errors as `{"error": "message"}`.

## Commands

```bash
go vet ./...                           # Static analysis
golangci-lint run ./...                # Lint (see .golangci.yml)
go test ./...                          # Run tests
gofmt -w .                             # Format all Go files
goose -dir db/migrations postgres "$DATABASE_URL" up   # Run migrations
docker compose up --build              # Run full stack
```

## Environment Variables

- `DATABASE_URL` — PostgreSQL connection string (required)
- `PORT` — HTTP port (default: 8080)
- `LOG_LEVEL` — Logging verbosity

## Git Conventions

- **Commits:** Conventional Commits format — `feat:`, `fix:`, `chore:`, `docs:`, `refactor:`, `test:`
- Always propose a plan before implementing non-trivial changes.

## Gotchas

- ON DELETE RESTRICT on all foreign keys — no cascading deletes
- License consistency trigger enforces `user_assignment.customer_id` must match `licenses.customer_id`
- JSONB fields (metadata, customizations) use GIN indexes
- Auth is explicitly deferred — not in MVP scope
