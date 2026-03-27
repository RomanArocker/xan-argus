# Asset Hardware Categories — Design Specification

## Overview

Extend the asset system with user-manageable hardware categories and per-category custom field definitions. Assets are always hardware (software is covered by licenses). Categories are global (shared across all customers) and carry typed field definitions that appear on asset forms.

## Goals

- Replace the static `assets.type` enum with a flexible, user-managed category system
- Allow custom field definitions per category (text, number, date, boolean)
- Keep field values optional (with `required` flag reserved for future use)
- Seed sensible defaults while allowing full customization

## Data Model

### New Tables

#### hardware_categories

Global lookup table for hardware types.

| Column | Type | Constraints |
|--------|------|-------------|
| id | UUID | PK, DEFAULT gen_random_uuid() |
| name | TEXT | NOT NULL, UNIQUE |
| description | TEXT | |
| created_at | TIMESTAMPTZ | NOT NULL, DEFAULT now() |
| updated_at | TIMESTAMPTZ | NOT NULL, DEFAULT now() |

#### category_field_definitions

Custom fields belonging to a category.

| Column | Type | Constraints |
|--------|------|-------------|
| id | UUID | PK, DEFAULT gen_random_uuid() |
| category_id | UUID | NOT NULL, FK → hardware_categories(id) ON DELETE CASCADE |
| name | TEXT | NOT NULL |
| field_type | TEXT | NOT NULL, CHECK (field_type IN ('text', 'number', 'date', 'boolean')) |
| required | BOOLEAN | NOT NULL, DEFAULT false |
| sort_order | INTEGER | NOT NULL, DEFAULT 0 |
| created_at | TIMESTAMPTZ | NOT NULL, DEFAULT now() |
| updated_at | TIMESTAMPTZ | NOT NULL, DEFAULT now() |

**Constraints:**
- UNIQUE on `(category_id, name)` — no duplicate field names per category
- `set_updated_at()` trigger on both tables

### Changes to `assets`

| Change | Details |
|--------|---------|
| Drop column | `type` (remove CHECK constraint) |
| Add column | `category_id UUID NULL FK → hardware_categories(id) ON DELETE SET NULL` |
| Add column | `field_values JSONB NOT NULL DEFAULT '{}'` |
| Add index | GIN index on `field_values` |
| Retain column | `metadata JSONB` — unchanged, keeps its existing GIN index |

**Column distinction:** `field_values` stores structured, validated data keyed by field definition UUIDs. `metadata` remains a freeform key-value bag for ad-hoc data (IP addresses, notes, etc.) that doesn't fit a category schema.

**Why ON DELETE SET NULL:** If a category is deleted, assets keep their field_values data intact but become uncategorized. Safer than RESTRICT (blocks deletes) or CASCADE (wipes assets).

## Migration Strategy

Single migration file:

1. Create `hardware_categories` table with triggers
2. Create `category_field_definitions` table with triggers
3. Add `category_id` and `field_values` columns to `assets`
4. Drop `type` column from `assets`
5. Delete any existing assets with `type = 'software'` (confirmed safe to drop)
6. Seed default categories: Laptop, Server, Printer, Monitor, Network Device

## API Endpoints

### Hardware Categories (global resource)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/hardware-categories` | List all categories |
| POST | `/api/v1/hardware-categories` | Create category |
| GET | `/api/v1/hardware-categories/{id}` | Get category with field definitions inline |
| PUT | `/api/v1/hardware-categories/{id}` | Update category name/description |
| DELETE | `/api/v1/hardware-categories/{id}` | Delete category (assets become uncategorized) |

### Field Definitions (nested under category)

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/hardware-categories/{id}/fields` | Add field definition |
| PUT | `/api/v1/hardware-categories/{id}/fields/{fieldId}` | Update field name/sort_order (field_type is immutable) |
| DELETE | `/api/v1/hardware-categories/{id}/fields/{fieldId}` | Remove field definition |

**Note:** `field_type` cannot be changed after creation — changing a type would silently invalidate existing field_values across all assets. To change a field's type, delete and recreate it.

### Asset Changes

- `POST/PUT` asset accepts `category_id` (nullable) and `field_values` (object)
- `GET` asset returns `category` object (with field definitions) alongside `field_values`
- `GET` list supports `?category_id=` query parameter for filtering

### Response Shapes

**GET /api/v1/hardware-categories/{id}:**
```json
{
  "id": "uuid",
  "name": "Laptop",
  "description": "Portable computers",
  "fields": [
    {
      "id": "uuid",
      "name": "RAM",
      "field_type": "number",
      "required": false,
      "sort_order": 0
    },
    {
      "id": "uuid",
      "name": "Serial Number",
      "field_type": "text",
      "required": false,
      "sort_order": 1
    }
  ],
  "created_at": "...",
  "updated_at": "..."
}
```

**GET /api/v1/assets/{id} (with category):**
```json
{
  "id": "uuid",
  "customer_id": "uuid",
  "category_id": "uuid",
  "category": {
    "id": "uuid",
    "name": "Laptop",
    "fields": [...]
  },
  "name": "ThinkPad T14s",
  "description": "Dev laptop",
  "field_values": {
    "<ram-field-uuid>": 16,
    "<serial-field-uuid>": "PF-4X9K2L"
  },
  "metadata": {},
  "created_at": "...",
  "updated_at": "..."
}
```

## Validation & Edge Cases

### field_values Validation (on asset create/update)

- Keys must be valid field definition UUIDs belonging to the asset's category
- Values must match the field's `field_type`:
  - `text` → string
  - `number` → numeric (int or float)
  - `date` → string in YYYY-MM-DD format
  - `boolean` → true/false
- Unknown keys are rejected with 400 Bad Request
- Missing keys are allowed (all fields optional in MVP)
- The `required` flag on field definitions is stored but **ignored during validation in MVP** — reserved for future enforcement

### Category Change

When an asset's `category_id` changes, `field_values` is cleared. Old values from a different category's schema are meaningless.

### Category Deletion

`ON DELETE SET NULL` on `assets.category_id`. Assets keep their `field_values` data but become uncategorized. The UI can display orphaned values as read-only.

### Field Definition Deletion

Deleting a field definition triggers a PostgreSQL `AFTER DELETE` trigger on `category_field_definitions` that strips that key from `field_values` across all affected assets:

```sql
CREATE FUNCTION strip_deleted_field_values() RETURNS trigger AS $$
BEGIN
    UPDATE assets
    SET field_values = field_values - OLD.id::text
    WHERE category_id = OLD.category_id;
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_strip_deleted_field_values
    AFTER DELETE ON category_field_definitions
    FOR EACH ROW EXECUTE FUNCTION strip_deleted_field_values();
```

This follows the project's PostgreSQL-first principle — the cleanup is enforced at the database level regardless of which client deletes the field.

### Constraints Summary

- Category names are globally unique
- Field names are unique per category: `UNIQUE(category_id, name)`
- No cascading deletes on assets — ON DELETE RESTRICT on `assets.customer_id`, ON DELETE SET NULL on `assets.category_id`

## Project Structure

New files following existing patterns:

```
internal/model/hardware_category.go
internal/repository/hardware_category.go
internal/handler/hardware_category.go
db/migrations/003_hardware_categories.sql
```

## Seed Data

Default categories seeded in migration:

| Name | Description |
|------|-------------|
| Laptop | Portable computers |
| Server | Rack-mounted or tower servers |
| Printer | Printers and multifunction devices |
| Monitor | Displays and screens |
| Network Device | Switches, routers, access points |
