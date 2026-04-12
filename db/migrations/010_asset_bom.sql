-- db/migrations/010_asset_bom.sql

-- +goose Up

-- 1. units table (no timestamps, no audit — pure reference data)
CREATE TABLE units (
    id   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE
);

-- 2. Seed units
INSERT INTO units (name) VALUES
    ('Piece'), ('Hour'), ('License'), ('Month'), ('Year'), ('kg'), ('m'), ('m²')
ON CONFLICT DO NOTHING;

-- 3. asset_bom_items table
CREATE TABLE asset_bom_items (
    id         UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    asset_id   UUID          NOT NULL REFERENCES assets(id) ON DELETE RESTRICT,
    name       TEXT          NOT NULL,
    quantity   NUMERIC(12,4) NOT NULL,
    unit_id    UUID          NOT NULL REFERENCES units(id) ON DELETE RESTRICT,
    unit_price NUMERIC(12,4) NOT NULL,
    currency   TEXT          NOT NULL CHECK (currency IN ('CHF', 'EUR', 'USD', 'GBP', 'JPY', 'CAD', 'AUD')),
    notes      TEXT,
    sort_order INT           NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ   NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

-- 4. updated_at trigger on asset_bom_items
CREATE TRIGGER trg_asset_bom_items_updated_at
    BEFORE UPDATE ON asset_bom_items
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- 5. Audit trigger on asset_bom_items
CREATE TRIGGER trg_asset_bom_items_audit
    AFTER INSERT OR UPDATE ON asset_bom_items
    FOR EACH ROW EXECUTE FUNCTION audit_trigger();

-- 6. Soft-delete guard on assets: block soft-delete if active BOM items exist

-- +goose StatementBegin
CREATE FUNCTION check_soft_delete_assets_bom() RETURNS TRIGGER AS $$
BEGIN
    IF NEW.deleted_at IS NOT NULL AND OLD.deleted_at IS NULL THEN
        IF EXISTS (
            SELECT 1 FROM asset_bom_items
            WHERE asset_id = OLD.id AND deleted_at IS NULL
        ) THEN
            RAISE EXCEPTION 'cannot delete asset with active BOM items'
                USING ERRCODE = '23503';
        END IF;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER trg_assets_soft_delete_bom_guard
    BEFORE UPDATE ON assets
    FOR EACH ROW EXECUTE FUNCTION check_soft_delete_assets_bom();

-- 7. Partial unique index: name unique per asset (active items only)
CREATE UNIQUE INDEX idx_asset_bom_items_name_active
    ON asset_bom_items(asset_id, name) WHERE deleted_at IS NULL;

-- +goose Down

DROP INDEX IF EXISTS idx_asset_bom_items_name_active;
DROP TRIGGER IF EXISTS trg_assets_soft_delete_bom_guard ON assets;
DROP FUNCTION IF EXISTS check_soft_delete_assets_bom();
DROP TRIGGER IF EXISTS trg_asset_bom_items_audit ON asset_bom_items;
DROP TRIGGER IF EXISTS trg_asset_bom_items_updated_at ON asset_bom_items;
DROP TABLE IF EXISTS asset_bom_items;
DROP TABLE IF EXISTS units;
