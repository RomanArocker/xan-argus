-- +goose Up

-- Services: no unique index exists yet
CREATE UNIQUE INDEX idx_services_name_active ON services(name) WHERE deleted_at IS NULL;

-- Users: composite natural key for import matching
CREATE UNIQUE INDEX idx_users_name_type_active ON users(first_name, last_name, type) WHERE deleted_at IS NULL;

-- Assets: composite natural key for import matching
CREATE UNIQUE INDEX idx_assets_name_customer_active ON assets(name, customer_id) WHERE deleted_at IS NULL;

-- Licenses: composite natural key for import matching
CREATE UNIQUE INDEX idx_licenses_product_customer_key_active ON licenses(product_name, customer_id, license_key) WHERE deleted_at IS NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_licenses_product_customer_key_active;
DROP INDEX IF EXISTS idx_assets_name_customer_active;
DROP INDEX IF EXISTS idx_users_name_type_active;
DROP INDEX IF EXISTS idx_services_name_active;
