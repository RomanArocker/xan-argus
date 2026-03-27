# Restrict User Assignments to Customer Staff Only

**Date:** 2026-03-27
**Status:** Draft

## Problem

The `user_assignments` table links users to customers, but currently accepts any user type — including `internal_staff`. Assigned users on a customer should only be `customer_staff` (employees of that company). Internal technician assignments to customers will be implemented separately in the future.

## Solution

Enforce the restriction at two layers:

1. **Database trigger** — safety net that prevents invalid data regardless of how it enters the system
2. **API validation** — provides clear, user-friendly error messages before hitting the database

## Database Layer

### New Migration

Add a trigger function on `user_assignments` that runs BEFORE INSERT and validates that the referenced user has `type = 'customer_staff'`.

```sql
-- +goose Up

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION check_user_assignment_type()
RETURNS TRIGGER AS $$
DECLARE
    user_type TEXT;
BEGIN
    SELECT type INTO user_type FROM users WHERE id = NEW.user_id;

    IF user_type IS NULL THEN
        RAISE EXCEPTION 'user_id % does not exist', NEW.user_id;
    END IF;

    IF user_type != 'customer_staff' THEN
        RAISE EXCEPTION 'only customer_staff users can be assigned to customers, got %', user_type;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER trg_user_assignments_type_check
    BEFORE INSERT ON user_assignments
    FOR EACH ROW EXECUTE FUNCTION check_user_assignment_type();

-- +goose Down
DROP TRIGGER IF EXISTS trg_user_assignments_type_check ON user_assignments;
DROP FUNCTION IF EXISTS check_user_assignment_type();
```

**Design decisions:**
- BEFORE INSERT only — `user_id` is immutable after creation (no UPDATE trigger needed)
- Pattern matches the existing `check_license_customer_consistency` trigger
- No UPDATE trigger on `users.type` changes — if a user's type changes to `internal_staff` while they have existing assignments, those assignments remain valid (historical data). This edge case can be addressed later if needed.

## API Layer

### Handler Changes

`UserAssignmentHandler` needs to validate the user type before calling the repository. Two options for accessing user data:

**Chosen approach:** Add a `GetUserType(ctx, userID)` helper method to `UserAssignmentRepository`. This avoids injecting a full `UserRepository` dependency for a single-field lookup, and keeps the query co-located with the assignment logic.

```go
func (r *UserAssignmentRepository) GetUserType(ctx context.Context, userID pgtype.UUID) (string, error) {
    var userType string
    err := r.pool.QueryRow(ctx,
        `SELECT type FROM users WHERE id = $1`, userID,
    ).Scan(&userType)
    if err != nil {
        return "", fmt.Errorf("getting user type: %w", err)
    }
    return userType, nil
}
```

In `UserAssignmentHandler.create()`, before calling `repo.Create()`:

```go
userType, err := h.repo.GetUserType(r.Context(), input.UserID)
if err != nil {
    if errors.Is(err, pgx.ErrNoRows) {
        writeError(w, http.StatusBadRequest, "user not found")
        return
    }
    writeError(w, http.StatusInternalServerError, "failed to validate user")
    return
}
if userType != "customer_staff" {
    writeError(w, http.StatusBadRequest, "only customer_staff users can be assigned to customers")
    return
}
```

**Error responses:**
- `400 Bad Request` — `"user not found"` (invalid user_id)
- `400 Bad Request` — `"only customer_staff users can be assigned to customers"` (wrong type)

**Trigger fallthrough behavior:** If the DB trigger fires despite API validation (e.g., a future code path bypasses the handler), the error falls through to the generic `500 "failed to create user assignment"` response. This is acceptable — the trigger is a safety net, and the API validation is the primary enforcement point.

## Files Changed

| File | Change |
|------|--------|
| `db/migrations/004_user_assignment_type_check.sql` | New migration with trigger |
| `internal/repository/user_assignment.go` | Add `GetUserType()` method |
| `internal/handler/user_assignment.go` | Add type validation in `create()` |

## Out of Scope

- Assigning internal technicians to customers (future feature)
- Restricting changes to `users.type` when assignments exist
- UI changes (API-only for now)
