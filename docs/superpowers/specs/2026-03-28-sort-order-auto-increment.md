# Sort Order Auto-Increment for Field Definitions

**Date:** 2026-03-28
**Status:** Approved

## Problem

When a new `category_field_definitions` record is created, `sort_order` defaults to `0`. Multiple fields in the same category end up with the same value, making display order non-deterministic (falls back to alphabetical name ordering only).

## Goal

New fields automatically receive a `sort_order` value one higher than the current maximum for that category — invisibly, with no UI change.

## Approach

Use a PostgreSQL subquery inside the INSERT to compute `sort_order` atomically:

```sql
INSERT INTO category_field_definitions (category_id, name, field_type, sort_order)
VALUES ($1, $2, $3,
  (SELECT COALESCE(MAX(sort_order), -1) + 1
   FROM category_field_definitions WHERE category_id = $1))
RETURNING id, category_id, name, field_type, required, sort_order, created_at, updated_at
```

- First field in a category: `COALESCE(NULL, -1) + 1 = 0`
- Each subsequent field: previous max + 1
- Atomically safe — no separate SELECT query, no race condition

## Scope

| Layer | File | Change |
|---|---|---|
| Repository | `internal/repository/hardware_category.go` | Replace static `sortOrder` value with subquery in `CreateField` |
| Model | `internal/model/hardware_category.go` | No change — `SortOrder *int` stays on input struct but is no longer used |
| Handler | `internal/handler/hardware_category.go` | No change |
| Templates | `web/templates/` | No change |

## Out of Scope

- Manual sort order editing via UI (deferred)
- Drag-and-drop reordering (deferred)
- Backfilling existing records with `sort_order = 0`
