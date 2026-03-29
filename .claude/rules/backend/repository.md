# Repository Conventions

Repositories live in `internal/repository/`. Each entity gets its own file and struct. Repositories own all SQL — no raw queries elsewhere.

## Structure

```go
type CustomerRepository struct {
    pool *pgxpool.Pool
}

func NewCustomerRepository(pool *pgxpool.Pool) *CustomerRepository {
    return &CustomerRepository{pool: pool}
}
```

- One struct per entity with a `pool *pgxpool.Pool` field
- Constructor: `NewEntityRepository(pool *pgxpool.Pool)`
- All methods take `ctx context.Context` as first parameter

## CRUD Method Signatures

Follow these exact signatures for consistency:

```go
func (r *EntityRepo) Create(ctx context.Context, input model.CreateEntityInput) (model.Entity, error)
func (r *EntityRepo) GetByID(ctx context.Context, id pgtype.UUID) (model.Entity, error)
func (r *EntityRepo) List(ctx context.Context, params model.ListParams) ([]model.Entity, error)
func (r *EntityRepo) Update(ctx context.Context, id pgtype.UUID, input model.UpdateEntityInput) (model.Entity, error)
func (r *EntityRepo) Delete(ctx context.Context, id pgtype.UUID) error
```

Additional query methods use descriptive names: `ListByCustomer()`, `ListByCategory()`, `ListFields()`.

## SQL Patterns

### INSERT with RETURNING

```go
func (r *CustomerRepository) Create(ctx context.Context, input model.CreateCustomerInput) (model.Customer, error) {
    var c model.Customer
    err := r.pool.QueryRow(ctx,
        `INSERT INTO customers (name, contact_email, notes)
         VALUES ($1, $2, $3)
         RETURNING id, name, contact_email, notes, created_at, updated_at`,
        input.Name, input.ContactEmail, input.Notes,
    ).Scan(&c.ID, &c.Name, &c.ContactEmail, &c.Notes, &c.CreatedAt, &c.UpdatedAt)
    if err != nil {
        return c, fmt.Errorf("creating customer: %w", err)
    }
    return c, nil
}
```

### SELECT single row

```go
err := r.pool.QueryRow(ctx,
    `SELECT id, name, contact_email, notes, created_at, updated_at
     FROM customers WHERE id = $1 AND deleted_at IS NULL`, id,
).Scan(&c.ID, &c.Name, &c.ContactEmail, &c.Notes, &c.CreatedAt, &c.UpdatedAt)
```

**Every read query must include `AND deleted_at IS NULL`** (or `WHERE deleted_at IS NULL` if no other conditions).

### SELECT multiple rows with search

```go
func (r *CustomerRepository) List(ctx context.Context, params model.ListParams) ([]model.Customer, error) {
    params.Normalize()
    query := `SELECT id, name, contact_email, notes, created_at, updated_at FROM customers WHERE deleted_at IS NULL`
    args := []any{}
    if params.Search != "" {
        query += ` AND name ILIKE $1`
        args = append(args, "%"+params.Search+"%")
    }
    query += fmt.Sprintf(` ORDER BY name LIMIT $%d OFFSET $%d`, len(args)+1, len(args)+2)
    args = append(args, params.Limit, params.Offset)
    rows, err := r.pool.Query(ctx, query, args...)
    if err != nil {
        return nil, fmt.Errorf("listing customers: %w", err)
    }
    defer rows.Close()
    return pgx.CollectRows(rows, pgx.RowToStructByPos[model.Customer])
}
```

Key points:
- Use positional parameters: `$1, $2, $3...`
- Dynamic parameter numbering with `len(args)+1`
- Search uses `ILIKE` with `%` wildcards
- Use `pgx.CollectRows` with `pgx.RowToStructByPos` for result sets
- Always `defer rows.Close()`

### UPDATE with COALESCE

```go
err := r.pool.QueryRow(ctx,
    `UPDATE customers SET
        name = COALESCE($2, name),
        contact_email = COALESCE($3, contact_email),
        notes = COALESCE($4, notes)
     WHERE id = $1 AND deleted_at IS NULL
     RETURNING id, name, contact_email, notes, created_at, updated_at`,
    id, input.Name, input.ContactEmail, input.Notes,
).Scan(...)
```

`COALESCE` allows partial updates — only non-nil fields are changed. This works because `*string` input fields marshal as `null` when nil.

**COALESCE limitation for nullable columns:** `COALESCE(NULL, column)` keeps the old value — you can never clear a nullable field this way. Go's JSON decoder also cannot distinguish between "field absent" and "field is null" for pointer types (both result in nil). For nullable columns that users need to clear (dates, optional FKs, optional text), use direct assignment instead of COALESCE:

```go
// Clearable nullable fields: direct assignment (NULL clears the value)
valid_from = $6,
valid_until = $7,
// Required fields: COALESCE (NULL keeps old value for partial updates)
product_name = COALESCE($3, product_name),
```

When using direct assignment for nullable fields, the form must always send all fields (with `null` for empty optionals).

### Soft DELETE with RowsAffected check

```go
func (r *CustomerRepository) Delete(ctx context.Context, id pgtype.UUID) error {
    result, err := r.pool.Exec(ctx,
        `UPDATE customers SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL`, id)
    if err != nil {
        return fmt.Errorf("deleting customer: %w", err)
    }
    if result.RowsAffected() == 0 {
        return fmt.Errorf("customer not found")
    }
    return nil
}
```

**No hard deletes** — all Delete methods use `UPDATE SET deleted_at = now()`. Soft-delete guard triggers (PostgreSQL) enforce referential integrity by raising `ERRCODE '23503'` if active dependents exist.

## Error Handling

- Always wrap errors with context: `fmt.Errorf("creating customer: %w", err)`
- Use verb + entity as context: `creating`, `getting`, `listing`, `updating`, `deleting`
- Never swallow errors silently
- Let PostgreSQL errors (FK violations, unique constraints) bubble up to handlers — don't interpret them here

## Do / Don't

- **Do** use `RETURNING` clauses to return the full entity after INSERT/UPDATE
- **Do** use positional `$N` parameters — never string interpolation
- **Do** use `pgx.CollectRows` with `RowToStructByPos` for lists
- **Do** check `RowsAffected()` for soft DELETE operations
- **Do** include `AND deleted_at IS NULL` in every SELECT, UPDATE, and soft DELETE query
- **Don't** use hard `DELETE FROM` — always soft delete via `UPDATE SET deleted_at = now()`
- **Don't** interpret PG error codes in repositories — that's the handler's job
- **Don't** use `SELECT *` — always list columns explicitly
- **Don't** add business logic — repos are pure data access
- **Don't** import `net/http` — repos know nothing about HTTP
