# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

XAN-Argus is a multi-tenant asset/user/license/service management system for an IT services business. MVP scope: CRUD, search, and overview — no authentication.

Design spec: `@docs/superpowers/specs/2026-03-27-xan-argus-design.md`
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
- **Soft delete + audit trail:** no hard deletes — all tables use `deleted_at TIMESTAMPTZ` column. All read queries must filter with `WHERE deleted_at IS NULL`. Audit log via PostgreSQL triggers (append-only `audit_log` table). See `docs/superpowers/specs/2026-03-29-soft-delete-audit-trail-design.md`.
- **Minimal dependencies:** prefer stdlib over third-party packages wherever reasonable.
- **No service layer:** handlers call repositories directly in MVP.
- **Minimal UI effort:** semantic HTML + HTMX + existing `style.css` — no CSS frameworks.

Detailed conventions for each layer are in `.claude/rules/` (auto-loaded):
- `backend/` — handlers, repository, models, database/migrations
- `frontend/` — templates, HTMX, styling
- `shared/` — naming conventions across all layers

## Commands

```bash
go vet ./...                           # Static analysis
golangci-lint run ./...                # Lint (see .golangci.yml)
go test ./...                          # Run tests
gofmt -w .                             # Format all Go files
goose -dir db/migrations postgres "$DATABASE_URL" up   # Run migrations
docker compose up --build              # Run full stack
```

## Code Exploration

When exploring Go files to understand structure (not to edit), prefer AST-based tools over full reads:

- **`smart_outline(file_path)`** — shows all functions, types, methods with signatures (bodies folded). Use this first.
- **`smart_search(query)`** — finds functions/types by name across the codebase.
- **`smart_unfold(file_path, symbol_name)`** — expands a single function body when needed.

Only use `Read` on a Go file when you need to **edit** it (Edit tool requires content in context).

Quick reference for this project:
- Handlers: `internal/handler/*.go`
- Repositories: `internal/repository/*.go`
- Models: `internal/model/*.go`
- Entry point: `cmd/server/main.go`

## Environment Variables

- `DATABASE_URL` — PostgreSQL connection string (required)
- `PORT` — HTTP port (default: 8080)
- `LOG_LEVEL` — Logging verbosity

## Git Conventions

- **Commits:** Conventional Commits format — `feat:`, `fix:`, `chore:`, `docs:`, `refactor:`, `test:`
- Always propose a plan before implementing non-trivial changes.
- **Push reminder:** After completing a feature, bugfix, or any meaningful chunk of work, remind the user to push to GitHub. Do NOT push automatically — wait for explicit approval.

## Gotchas

- Auth is explicitly deferred — not in MVP scope
- **Soft delete filtering is mandatory:** every new read query (GetByID, List, custom) MUST include `WHERE deleted_at IS NULL` or soft-deleted records will reappear
- UNIQUE constraints use partial indexes (`WHERE deleted_at IS NULL`) — not plain `UNIQUE`
- See `.claude/rules/backend/database.md` for FK, constraint, and JSONB conventions
