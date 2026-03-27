-- db/migrations/003_hardware_categories.sql

-- +goose Up

-- hardware_categories
CREATE TABLE hardware_categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TRIGGER trg_hardware_categories_updated_at
    BEFORE UPDATE ON hardware_categories
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- category_field_definitions
CREATE TABLE category_field_definitions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    category_id UUID NOT NULL REFERENCES hardware_categories(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    field_type TEXT NOT NULL CHECK (field_type IN ('text', 'number', 'date', 'boolean')),
    required BOOLEAN NOT NULL DEFAULT false,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(category_id, name)
);

CREATE TRIGGER trg_category_field_definitions_updated_at
    BEFORE UPDATE ON category_field_definitions
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- strip_deleted_field_values trigger
-- +goose StatementBegin
CREATE FUNCTION strip_deleted_field_values() RETURNS trigger AS $$
BEGIN
    UPDATE assets
    SET field_values = field_values - OLD.id::text
    WHERE category_id = OLD.category_id;
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER trg_strip_deleted_field_values
    AFTER DELETE ON category_field_definitions
    FOR EACH ROW EXECUTE FUNCTION strip_deleted_field_values();

-- Modify assets table: remove type, add category_id and field_values
DELETE FROM assets WHERE type = 'software';

ALTER TABLE assets DROP COLUMN type;

ALTER TABLE assets ADD COLUMN category_id UUID REFERENCES hardware_categories(id) ON DELETE SET NULL;

ALTER TABLE assets ADD COLUMN field_values JSONB NOT NULL DEFAULT '{}';

CREATE INDEX idx_assets_field_values ON assets USING GIN (field_values);

-- Seed default categories
INSERT INTO hardware_categories (name, description) VALUES
    ('Laptop', 'Portable computers'),
    ('Server', 'Rack-mounted or tower servers'),
    ('Printer', 'Printers and multifunction devices'),
    ('Monitor', 'Displays and screens'),
    ('Network Device', 'Switches, routers, access points');

-- +goose Down
ALTER TABLE assets DROP COLUMN IF EXISTS field_values;
ALTER TABLE assets DROP COLUMN IF EXISTS category_id;
ALTER TABLE assets ADD COLUMN type TEXT NOT NULL DEFAULT 'hardware' CHECK (type IN ('hardware', 'software'));
DROP TRIGGER IF EXISTS trg_strip_deleted_field_values ON category_field_definitions;
DROP FUNCTION IF EXISTS strip_deleted_field_values;
DROP TABLE IF EXISTS category_field_definitions;
DROP TABLE IF EXISTS hardware_categories;
