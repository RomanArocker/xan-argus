# Project Learnings

> Auto-generated. Read this file at session start to avoid repeating past mistakes.

## Bugs & Fixes

| Date | Problem | Root Cause | Solution |
|------|---------|------------|----------|
| 2026-03-27 | Goose migration failed with "unterminated dollar-quoted string" on PL/pgSQL functions | Goose splits SQL on `;` by default — `$$` blocks with `;` inside get split incorrectly | Wrap PL/pgSQL functions with `-- +goose StatementBegin` / `-- +goose StatementEnd` |
| 2026-03-27 | All list pages rendered the wrong template (services list instead of customers) | Go `template.ParseFiles` loads all templates into one namespace — multiple files defining `{{define "content"}}` overwrite each other, last one wins | Parse each page separately: layout + partials + one page file per `template.Template` instance. Split into `RenderPage` (layout-wrapped) and `RenderPartial` (standalone) |
| 2026-03-27 | Docker build failed: `go.mod requires go >= 1.26.1 (running go 1.23.12)` | Dockerfile used `golang:1.23-alpine` but local Go and go.mod are 1.26.1 | Update Dockerfile to `golang:1.26-alpine` |
| 2026-03-27 | App started but templates/migrations not found in Docker container | Dockerfile only copied the Go binary, not `db/migrations/` or `web/` | Add `COPY --from=builder /app/db/migrations /db/migrations` and `COPY --from=builder /app/web /web` to Dockerfile |

## Gotchas & Configuration

- **2026-03-27** — Go `html/template` with `ParseFiles`: when multiple files define the same template name (e.g., `"content"`), only the last parsed file's definition survives. Solution: create separate `*template.Template` instances per page, each combining layout + partials + one page file.
- **2026-03-27** — Goose requires `-- +goose StatementBegin` / `-- +goose StatementEnd` around any SQL containing semicolons within the statement body (PL/pgSQL functions, DO blocks). Without these markers, goose splits on `;` and sends incomplete SQL.
- **2026-03-27** — PostgreSQL `ON DELETE RESTRICT` means you must delete dependent records before parent records. Delete handlers should check for FK violation (pgconn error code `23503`) and return 409 Conflict instead of generic 404.
- **2026-03-27** — `pgx.CollectRows` with `pgx.RowToStructByPos` requires struct fields to match SELECT column order exactly. `pgtype.UUID`, `pgtype.Text`, `pgtype.Date` handle nullable columns.

## Workflow Insights

- **2026-03-27** — For HTMX live search, use `list_rows.html` partials that define a named template (e.g., `"customer_rows"`). The list page includes it inline; the `/rows` endpoint renders just the partial for `hx-swap`.
- **2026-03-27** — HTMX `json-enc` extension is needed when forms need to POST JSON to a REST API. Vendor it alongside `htmx.min.js` in `web/static/js/`.
- **2026-03-27** — Business language decision (English vs German) should be set in CLAUDE.md early — translating UI labels retroactively across a full plan is tedious.
