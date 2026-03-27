# Restrict User Assignments to Customer Staff — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Prevent `internal_staff` users from being assigned to customers via `user_assignments`, enforced at both DB and API layers.

**Architecture:** New DB trigger rejects non-`customer_staff` inserts into `user_assignments`. Handler validates user type before insert, returning a clear 400 error. Repository gets a `GetUserType()` helper.

**Tech Stack:** Go stdlib + pgx, PostgreSQL trigger, goose migration

**Spec:** `docs/superpowers/specs/2026-03-27-restrict-user-assignment-type-design.md`

---

## File Structure

| File | Action | Responsibility |
|------|--------|----------------|
| `db/migrations/004_user_assignment_type_check.sql` | Create | DB trigger enforcing customer_staff-only |
| `internal/repository/user_assignment.go` | Modify | Add `GetUserType()` method |
| `internal/repository/user_assignment_test.go` | Modify | Fix existing test (wrong user type), add trigger + GetUserType tests |
| `internal/handler/user_assignment.go` | Modify | Add type validation in `create()` |

---

### Task 1: Add Database Migration

**Files:**
- Create: `db/migrations/004_user_assignment_type_check.sql`

- [ ] **Step 1: Create the migration file**

```sql
-- db/migrations/004_user_assignment_type_check.sql

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

- [ ] **Step 2: Verify migration applies**

Run: `docker compose up --build`
Expected: Container starts, migrations complete, logs show "Migrations completed"

- [ ] **Step 3: Commit**

```bash
git add db/migrations/004_user_assignment_type_check.sql
git commit -m "feat: add trigger to restrict user_assignments to customer_staff only"
```

---

### Task 2: Fix Existing Test and Add Trigger Tests

The existing `user_assignment_test.go` creates a user with `Type: "employee"` which is invalid per the schema CHECK constraint (`customer_staff | internal_staff`). After the new trigger, this must use `customer_staff`. Add new test cases for the trigger.

**Files:**
- Modify: `internal/repository/user_assignment_test.go`

- [ ] **Step 1: Fix the existing test's user type**

In `TestUserAssignmentCRUD`, change the user creation from:
```go
user, err := userRepo.Create(ctx, model.CreateUserInput{Type: "employee", FirstName: "UA", LastName: "TestUser"})
```
to:
```go
user, err := userRepo.Create(ctx, model.CreateUserInput{Type: "customer_staff", FirstName: "UA", LastName: "TestUser"})
```

- [ ] **Step 2: Run existing test to verify it still passes**

Run: `docker compose up --build -d && docker compose exec app go test ./internal/repository/ -run TestUserAssignmentCRUD -v`
Expected: PASS

- [ ] **Step 3: Add test for internal_staff rejection**

Add a new test function below `TestUserAssignmentCRUD`:

```go
func TestUserAssignmentRejectsInternalStaff(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()

	customerRepo := repository.NewCustomerRepository(pool)
	customer, err := customerRepo.Create(ctx, model.CreateCustomerInput{Name: "Type Check Customer"})
	if err != nil {
		t.Fatalf("Create customer: %v", err)
	}
	t.Cleanup(func() { customerRepo.Delete(ctx, customer.ID) }) //nolint:errcheck

	userRepo := repository.NewUserRepository(pool)
	internalUser, err := userRepo.Create(ctx, model.CreateUserInput{Type: "internal_staff", FirstName: "Internal", LastName: "User"})
	if err != nil {
		t.Fatalf("Create internal user: %v", err)
	}
	t.Cleanup(func() { userRepo.Delete(ctx, internalUser.ID) }) //nolint:errcheck

	repo := repository.NewUserAssignmentRepository(pool)
	_, err = repo.Create(ctx, model.CreateUserAssignmentInput{
		UserID:     internalUser.ID,
		CustomerID: customer.ID,
		Role:       "admin",
	})
	if err == nil {
		t.Fatal("expected error when assigning internal_staff to customer, got nil")
	}
}
```

- [ ] **Step 4: Run the new test**

Run: `docker compose exec app go test ./internal/repository/ -run TestUserAssignmentRejectsInternalStaff -v`
Expected: PASS (trigger rejects the insert)

- [ ] **Step 5: Commit**

```bash
git add internal/repository/user_assignment_test.go
git commit -m "test: fix user type in assignment test, add internal_staff rejection test"
```

---

### Task 3: Add GetUserType Repository Method

**Files:**
- Modify: `internal/repository/user_assignment.go`

- [ ] **Step 1: Add the GetUserType method**

Add at the end of the file, before the closing of the package:

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

- [ ] **Step 2: Add test for GetUserType**

Add to `internal/repository/user_assignment_test.go`:

```go
func TestGetUserType(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()

	userRepo := repository.NewUserRepository(pool)

	customerStaff, err := userRepo.Create(ctx, model.CreateUserInput{Type: "customer_staff", FirstName: "CS", LastName: "User"})
	if err != nil {
		t.Fatalf("Create customer_staff: %v", err)
	}
	t.Cleanup(func() { userRepo.Delete(ctx, customerStaff.ID) }) //nolint:errcheck

	internalStaff, err := userRepo.Create(ctx, model.CreateUserInput{Type: "internal_staff", FirstName: "IS", LastName: "User"})
	if err != nil {
		t.Fatalf("Create internal_staff: %v", err)
	}
	t.Cleanup(func() { userRepo.Delete(ctx, internalStaff.ID) }) //nolint:errcheck

	repo := repository.NewUserAssignmentRepository(pool)

	typ, err := repo.GetUserType(ctx, customerStaff.ID)
	if err != nil {
		t.Fatalf("GetUserType customer_staff: %v", err)
	}
	if typ != "customer_staff" {
		t.Errorf("got %q, want %q", typ, "customer_staff")
	}

	typ, err = repo.GetUserType(ctx, internalStaff.ID)
	if err != nil {
		t.Fatalf("GetUserType internal_staff: %v", err)
	}
	if typ != "internal_staff" {
		t.Errorf("got %q, want %q", typ, "internal_staff")
	}
}
```

- [ ] **Step 3: Run tests**

Run: `docker compose exec app go test ./internal/repository/ -run "TestGetUserType|TestUserAssignment" -v`
Expected: All PASS

- [ ] **Step 4: Commit**

```bash
git add internal/repository/user_assignment.go internal/repository/user_assignment_test.go
git commit -m "feat: add GetUserType method to UserAssignmentRepository"
```

---

### Task 4: Add API Validation in Handler

**Files:**
- Modify: `internal/handler/user_assignment.go`

- [ ] **Step 1: Add type validation to the create method**

In `internal/handler/user_assignment.go`, in the `create()` method, add the following block **after** the `input.Role == ""` check and **before** calling `h.repo.Create()`:

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

Note: The `errors` and `pgx` imports are already present in this file.

- [ ] **Step 2: Verify the project compiles**

Run: `docker compose exec app go vet ./...`
Expected: No errors

- [ ] **Step 3: Run all tests**

Run: `docker compose exec app go test ./... -v`
Expected: All PASS

- [ ] **Step 4: Manual smoke test via API**

Create an internal_staff user and try to assign them:

```bash
# Create an internal_staff user
curl -s -X POST http://localhost:8080/api/v1/users \
  -H "Content-Type: application/json" \
  -d '{"type":"internal_staff","first_name":"Test","last_name":"Tech"}' | jq .

# Note the user ID from above, then try to assign (use any valid customer ID)
# Expected: 400 "only customer_staff users can be assigned to customers"
curl -s -X POST http://localhost:8080/api/v1/customers/{CUSTOMER_ID}/user-assignments \
  -H "Content-Type: application/json" \
  -d '{"user_id":"{USER_ID}","role":"admin"}' | jq .
```

- [ ] **Step 5: Commit**

```bash
git add internal/handler/user_assignment.go
git commit -m "feat: validate user type in user assignment handler"
```
