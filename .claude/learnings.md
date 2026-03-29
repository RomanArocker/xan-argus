# Project Learnings

> Auto-generated. Read this file at session start to avoid repeating past mistakes.

## Bugs & Fixes

| Date | Problem | Root Cause | Solution |
|------|---------|------------|----------|
| 2026-03-27 | Goose migration failed with "unterminated dollar-quoted string" on PL/pgSQL functions | Goose splits SQL on `;` by default — `$$` blocks with `;` inside get split incorrectly | Wrap PL/pgSQL functions with `-- +goose StatementBegin` / `-- +goose StatementEnd` |
| 2026-03-27 | All list pages rendered the wrong template (services list instead of customers) | Go `template.ParseFiles` loads all templates into one namespace — multiple files defining `{{define "content"}}` overwrite each other, last one wins | Parse each page separately: layout + partials + one page file per `template.Template` instance. Split into `RenderPage` (layout-wrapped) and `RenderPartial` (standalone) |
| 2026-03-27 | Docker build failed: `go.mod requires go >= 1.26.1 (running go 1.23.12)` | Dockerfile used `golang:1.23-alpine` but local Go and go.mod are 1.26.1 | Update Dockerfile to `golang:1.26-alpine` |
| 2026-03-27 | App started but templates/migrations not found in Docker container | Dockerfile only copied the Go binary, not `db/migrations/` or `web/` | Add `COPY --from=builder /app/db/migrations /db/migrations` and `COPY --from=builder /app/web /web` to Dockerfile |

| 2026-03-28 | Docker app container crashed on restart: DNS resolution for `db` failed | Stale Docker network from old `xan-pythia` containers still running — ports 5432/8080 occupied | Stop and remove old containers (`docker stop/rm xan-pythia-*`), then `docker compose down -v && docker compose up --build -d` for clean network |
| 2026-03-28 | Optional date field in `field_values` always failed validation with "must be in YYYY-MM-DD format" even when left blank | `validateFieldValues` checked `time.Parse` on the value without first checking for empty string — empty string from form is valid for an optional date field | Skip date format validation when the string value is `""` |
| 2026-03-28 | License form: empty optional fields (dates, license_key) caused "invalid request body" | `hx-ext="json-enc"` sends empty strings `""` for empty inputs — `pgtype.Date`/`pgtype.UUID` cannot deserialize `""` | Use custom JS `fetch()` instead of `json-enc` for forms with optional fields; send `null` for empty values |
| 2026-03-28 | License edit: clearing a date field had no effect (old value persisted) | `COALESCE(NULL, column)` keeps the old value — nullable fields can never be cleared with COALESCE. Go JSON also can't distinguish absent vs null for pointer types | Remove COALESCE for nullable columns (direct assignment `column = $N`); form always sends all fields with `null` for empty optionals |

## Failed Approaches

| Date | Attempted Approach | Why It Failed | What Worked Instead |
|------|-------------------|---------------|---------------------|
| 2026-03-28 | `git add` with paths that are in `.gitignore` (e.g., `.claude/`, `CLAUDE.md`) chained with `&&` to `git commit` | `git add` exits with code 1 when encountering ignored files, which aborts the entire `&&` chain — commit never runs | Exclude ignored paths from `git add` command; use `git rm --cached` separately to untrack them |

## Gotchas & Configuration

- **2026-03-30** — The `internal/importer/` package is the one exception to the "no service layer" rule. It uses dynamic SQL (`fmt.Sprintf` for table/column names) which is safe because all values come from hardcoded `EntityConfig` structs, never user input. Values always use `$N` positional params.
- **2026-03-30** — CSV exports must prepend UTF-8 BOM (`0xEF, 0xBB, 0xBF`) for Excel compatibility. On import, strip BOM before parsing. Go's `encoding/csv` does not handle BOM automatically.
- **2026-03-30** — For upsert match keys, every combination needs a partial unique index (`WHERE deleted_at IS NULL`). Missing indexes: services(name), users(first_name, last_name, type), assets(name, customer_id), licenses(product_name, customer_id, license_key) — added in migration 009.
- **2026-03-30** — Customer names ARE unique among active records (partial unique index), but NOT globally unique (soft-deleted duplicates can exist). FK name-based lookup works because all queries filter `WHERE deleted_at IS NULL`.
- **2026-03-29** — With soft delete, UNIQUE constraints must use partial indexes (`WHERE deleted_at IS NULL`) instead of plain `UNIQUE`. Otherwise a soft-deleted record blocks re-creation of the same name. The auto-generated PostgreSQL constraint name follows the pattern `tablename_column_key` (e.g., `customers_name_key`) — use this in `ALTER TABLE DROP CONSTRAINT`.
- **2026-03-29** — Soft-delete guard triggers use `ERRCODE = '23503'` (same as FK violation), so the existing `isFKViolation()` handler code catches them without changes. This works because pgx surfaces PG error codes regardless of statement type (UPDATE vs DELETE).
- **2026-03-29** — Docker DB user for psql access is `xanargus` (not `postgres` or `xan_argus`). Use: `docker compose exec db psql -U xanargus -d xanargus`.
- **2026-03-29** — `.claude/rules/` files are in `.gitignore` — use `git add -f` to force-add them when they need to be committed.
- **2026-03-28** — `ON CONFLICT DO NOTHING` (bare, no target) catches *any* constraint violation silently — including unexpected UNIQUE violations on non-PK columns. Always use `ON CONFLICT (id) DO NOTHING` to be explicit about which conflict you're tolerating.
- **2026-03-28** — When seeding data that references rows from a previous migration without fixed UUIDs (e.g., `hardware_categories` from migration 003), use `(SELECT id FROM table WHERE name = '...')` subqueries in the `VALUES` clause. Works because the `name` column has a UNIQUE constraint.
- **2026-03-27** — Go `html/template` with `ParseFiles`: when multiple files define the same template name (e.g., `"content"`), only the last parsed file's definition survives. Solution: create separate `*template.Template` instances per page, each combining layout + partials + one page file.
- **2026-03-27** — Goose requires `-- +goose StatementBegin` / `-- +goose StatementEnd` around any SQL containing semicolons within the statement body (PL/pgSQL functions, DO blocks). Without these markers, goose splits on `;` and sends incomplete SQL.
- **2026-03-27** — PostgreSQL `ON DELETE RESTRICT` means you must delete dependent records before parent records. Delete handlers should check for FK violation (pgconn error code `23503`) and return 409 Conflict instead of generic 404.
- **2026-03-27** — `pgx.CollectRows` with `pgx.RowToStructByPos` requires struct fields to match SELECT column order exactly. `pgtype.UUID`, `pgtype.Text`, `pgtype.Date` handle nullable columns.

- **2026-03-28** — When renaming a project (module, DB credentials, binary name), always do `docker compose down -v` to drop old volumes — DB name/user changed so old volume is incompatible.
- **2026-03-28** — After updating `.gitignore`, old containers from the previous project name may still be running and blocking ports. Check `docker ps` for orphaned containers before starting the new stack.

## Workflow Insights

- **2026-03-28** — Go versioning via ldflags: use `var version = "dev"` and `var gitCommit = "unknown"` (NOT `const`) as package-level vars in `main.go`. ldflags can only override `var`, not `const`. Pattern: `go build -ldflags "-X main.version=1.0.0 -X main.gitCommit=abc1234"`.
- **2026-03-28** — `TemplateEngine.RenderPage` accepts `data any` but all callers pass `map[string]any`. Safe to tighten the signature to `map[string]any` — no callers need updating. Enables adding a nil guard and automatic key injection (e.g. `data["Version"]`).
- **2026-03-28** — GnuWin32 Make (`winget install GnuWin32.Make`) installs to `C:\Program Files (x86)\GnuWin32\bin\` — this path is NOT automatically added to PATH. Requires a new shell session or manual PATH update before `make` works in Git Bash.
- **2026-03-28** — PostgreSQL BEFORE INSERT/UPDATE triggers (like `check_asset_customer_consistency`) fire before FK constraint checks. A `RAISE EXCEPTION` from a trigger produces SQLSTATE `P0001` (raise_exception), NOT `23503` (FK violation). So `isFKViolation()` won't catch trigger errors — they bubble up as generic 500s. This matches the license pattern and is by design. **Update 2026-03-29:** You CAN make triggers return a specific ERRCODE with `USING ERRCODE = '23503'` — the soft-delete guard triggers use this to emulate FK violations.
- **2026-03-27** — For HTMX live search, use `list_rows.html` partials that define a named template (e.g., `"customer_rows"`). The list page includes it inline; the `/rows` endpoint renders just the partial for `hx-swap`.
- **2026-03-27** — HTMX `json-enc` extension is needed when forms need to POST JSON to a REST API. Vendor it alongside `htmx.min.js` in `web/static/js/`.
- **2026-03-30** — Subagent-driven development works well for mechanical tasks with clear specs (Tasks 1-5, 8-10). The import engine (Task 6) and exporter (Task 7) were more complex but still succeeded with detailed prompts. Batching tightly coupled tasks (1-5 together) avoids compilation issues from missing dependencies.
- **2026-03-27** — Business language decision (English vs German) should be set in CLAUDE.md early — translating UI labels retroactively across a full plan is tedious.
