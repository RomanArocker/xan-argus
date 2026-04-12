# Assign User: Filter by Type and Existing Assignments

**Date:** 2026-04-12
**Status:** Approved

## Problem

The "Assign User" dropdown on the customer detail page currently shows all users regardless of type (`customer_staff` and `internal_staff`). Internal staff users serve a different purpose and should not appear in this context. Additionally, users already assigned to the customer should not appear as options, since the unique constraint `(user_id, customer_id)` would reject the assignment anyway.

## Solution

Add a new repository method `ListAvailableForCustomer` that returns only `customer_staff` users not yet assigned to the given customer. Update the page handler to use this method instead of the generic `List`.

## Changes

### 1. Repository — `internal/repository/user.go`

New method:

```go
func (r *UserRepository) ListAvailableForCustomer(ctx context.Context, customerID pgtype.UUID) ([]model.User, error)
```

SQL:

```sql
SELECT id, type, first_name, last_name, created_at, updated_at
FROM users
WHERE type = 'customer_staff'
  AND deleted_at IS NULL
  AND id NOT IN (
    SELECT user_id FROM user_assignments
    WHERE customer_id = $1 AND deleted_at IS NULL
  )
ORDER BY first_name, last_name
```

No pagination — the list of available users per customer is expected to remain small.

### 2. Page Handler — `internal/handler/page.go`

In `customerDetail`, replace:

```go
h.userRepo.List(r.Context(), model.ListParams{Limit: 100})
```

with:

```go
h.userRepo.ListAvailableForCustomer(r.Context(), id)
```

The template data key `"Users"` remains unchanged. No template modifications required.

## Scope

- Only affects the "Assign User" form on the customer detail page (`GET /customers/{id}`)
- No database schema changes
- No migrations required
- The `/users` list page and all other user-loading contexts are unaffected

## Out of Scope

- Filtering internal staff from other contexts (deferred — internal staff have a different future purpose)
- Search/autocomplete within the assign dropdown
- Authentication or role-based access control
