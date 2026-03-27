-- db/migrations/002_dependent_tables.sql

-- +goose Up

-- user_assignments
CREATE TABLE user_assignments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE RESTRICT,
    role TEXT NOT NULL,
    email TEXT,
    phone TEXT,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(user_id, customer_id)
);

CREATE TRIGGER trg_user_assignments_updated_at
    BEFORE UPDATE ON user_assignments
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- assets
CREATE TABLE assets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE RESTRICT,
    type TEXT NOT NULL CHECK (type IN ('hardware', 'software')),
    name TEXT NOT NULL,
    description TEXT,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_assets_metadata ON assets USING GIN (metadata);

CREATE TRIGGER trg_assets_updated_at
    BEFORE UPDATE ON assets
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- licenses
CREATE TABLE licenses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE RESTRICT,
    user_assignment_id UUID REFERENCES user_assignments(id) ON DELETE RESTRICT,
    product_name TEXT NOT NULL,
    license_key TEXT,
    quantity INTEGER NOT NULL DEFAULT 1,
    valid_from DATE,
    valid_until DATE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TRIGGER trg_licenses_updated_at
    BEFORE UPDATE ON licenses
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- License consistency trigger: ensure user_assignment belongs to same customer
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION check_license_customer_consistency()
RETURNS TRIGGER AS $$
DECLARE
    assignment_customer_id UUID;
BEGIN
    IF NEW.user_assignment_id IS NOT NULL THEN
        SELECT customer_id INTO assignment_customer_id
        FROM user_assignments
        WHERE id = NEW.user_assignment_id;

        IF assignment_customer_id IS NULL THEN
            RAISE EXCEPTION 'user_assignment_id % does not exist', NEW.user_assignment_id;
        END IF;

        IF assignment_customer_id != NEW.customer_id THEN
            RAISE EXCEPTION 'license customer_id (%) does not match user_assignment customer_id (%)',
                NEW.customer_id, assignment_customer_id;
        END IF;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER trg_licenses_customer_consistency
    BEFORE INSERT OR UPDATE ON licenses
    FOR EACH ROW EXECUTE FUNCTION check_license_customer_consistency();

-- customer_services
CREATE TABLE customer_services (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE RESTRICT,
    service_id UUID NOT NULL REFERENCES services(id) ON DELETE RESTRICT,
    customizations JSONB DEFAULT '{}',
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(customer_id, service_id)
);

CREATE INDEX idx_customer_services_customizations ON customer_services USING GIN (customizations);

CREATE TRIGGER trg_customer_services_updated_at
    BEFORE UPDATE ON customer_services
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TABLE IF EXISTS customer_services;
DROP TABLE IF EXISTS licenses;
DROP TABLE IF EXISTS assets;
DROP TABLE IF EXISTS user_assignments;
DROP FUNCTION IF EXISTS check_license_customer_consistency;
