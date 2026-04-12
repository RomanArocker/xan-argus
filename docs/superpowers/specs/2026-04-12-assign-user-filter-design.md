# Assign User: Filter by Type and Existing Assignments

**Date:** 2026-04-12
**Status:** Approved

## Problem

The "Assign User" dropdown on the customer detail page currently shows all users regardless of type (`customer_staff` and `internal_staff`). Internal staff users serve a different purpose and should not appear in this context. Additionally, users already assigned to the customer should not appear as options, since the unique constraint `(user_id, customer_id)` would reject the assignment anyway.

## Solution

Add a new repository method `ListAvailableForCustomer` that returns only `customer_staff` users not yet assigned to the given customer. Update the page handler to use two separate user fetches: one for the dropdown (filtered), one for building display maps for existing assignments.

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

**Note on `NOT IN` safety:** `user_assignments.user_id` is `NOT NULL` by schema constraint, so the `NOT IN` subquery never produces `NULL` values and is safe from SQL three-valued logic issues.

### 2. Page Handler — `internal/handler/page.go`

In `customerDetail`, the existing `users` variable is used for **two distinct purposes**:
1. Populating the "Assign User" dropdown
2. Building display maps for already-assigned users shown in the assignment table (`buildAssignmentMap`, `buildUserAssignmentDisplayList`)

These must be split into two separate fetch calls:

```go
// For the dropdown: only unassigned customer_staff
availableUsers, _ := h.userRepo.ListAvailableForCustomer(r.Context(), id)

// For display maps: all customer_staff (only customer_staff can be assigned, so this covers all assignments)
allCustomerStaff, _ := h.userRepo.List(r.Context(), model.ListParams{Filter: "customer_staff", Limit: 500})
```

- `availableUsers` is passed to the template as `"Users"` (dropdown options)
- `allCustomerStaff` replaces the existing `users` variable in calls to `buildAssignmentMap` and `buildUserAssignmentDisplayList`

**Error handling:** Both calls follow the existing convention in `customerDetail` (errors silently ignored with `_`). No change to error handling behavior.

## Scope

- Only affects the "Assign User" form on the customer detail page (`GET /customers/{id}`)
- No database schema changes
- No migrations required
- The `/users` list page and all other user-loading contexts are unaffected

## Out of Scope

- Filtering internal staff from other contexts (deferred — internal staff have a different future purpose)
- Search/autocomplete within the assign dropdown
- Authentication or role-based access control
