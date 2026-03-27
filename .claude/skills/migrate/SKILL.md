---
name: migrate
description: Create or run goose database migrations. Use when schema changes are needed. Accepts arguments like "create add_users_table" or "up" or "status".
disable-model-invocation: true
---

## Database Migrations

Migrations live in `db/migrations/` and use goose with pure SQL.

### Usage

`/migrate $ARGUMENTS`

- `/migrate create <name>` — Create a new migration file: `goose -dir db/migrations create <name> sql`
- `/migrate up` — Run pending migrations: `goose -dir db/migrations postgres "$DATABASE_URL" up`
- `/migrate down` — Roll back last migration: `goose -dir db/migrations postgres "$DATABASE_URL" down`
- `/migrate status` — Show migration status: `goose -dir db/migrations postgres "$DATABASE_URL" status`

### Migration Guidelines

- Use pure SQL (not Go migrations)
- Always include both `-- +goose Up` and `-- +goose Down` sections
- Use `IF NOT EXISTS` / `IF EXISTS` for safety
- Follow the existing schema patterns: UUID primary keys, `created_at`/`updated_at` timestamps, `set_updated_at()` trigger
- Add CHECK constraints for enums, GIN indexes for JSONB columns
- Use ON DELETE RESTRICT for foreign keys
