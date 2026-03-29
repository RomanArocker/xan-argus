-- +goose Up

-- 1. Audit log table
CREATE TABLE audit_log (
    id BIGSERIAL PRIMARY KEY,
    table_name TEXT NOT NULL,
    record_id UUID NOT NULL,
    action TEXT NOT NULL CHECK (action IN ('INSERT', 'UPDATE', 'DELETE')),
    old_values JSONB,
    new_values JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_audit_log_record ON audit_log(table_name, record_id);
CREATE INDEX idx_audit_log_created ON audit_log(created_at);

-- 2. Generic audit trigger function
-- +goose StatementBegin
CREATE FUNCTION audit_trigger() RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        INSERT INTO audit_log(table_name, record_id, action, new_values)
        VALUES (TG_TABLE_NAME, NEW.id, 'INSERT', row_to_json(NEW)::jsonb);
    ELSIF TG_OP = 'UPDATE' THEN
        IF NEW.deleted_at IS NOT NULL AND OLD.deleted_at IS NULL THEN
            INSERT INTO audit_log(table_name, record_id, action, old_values)
            VALUES (TG_TABLE_NAME, OLD.id, 'DELETE', row_to_json(OLD)::jsonb);
        ELSE
            INSERT INTO audit_log(table_name, record_id, action, old_values, new_values)
            VALUES (TG_TABLE_NAME, OLD.id, 'UPDATE', row_to_json(OLD)::jsonb, row_to_json(NEW)::jsonb);
        END IF;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

-- 3. Add deleted_at to all 9 tables
ALTER TABLE customers ADD COLUMN deleted_at TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN deleted_at TIMESTAMPTZ;
ALTER TABLE services ADD COLUMN deleted_at TIMESTAMPTZ;
ALTER TABLE user_assignments ADD COLUMN deleted_at TIMESTAMPTZ;
ALTER TABLE assets ADD COLUMN deleted_at TIMESTAMPTZ;
ALTER TABLE licenses ADD COLUMN deleted_at TIMESTAMPTZ;
ALTER TABLE customer_services ADD COLUMN deleted_at TIMESTAMPTZ;
ALTER TABLE hardware_categories ADD COLUMN deleted_at TIMESTAMPTZ;
ALTER TABLE category_field_definitions ADD COLUMN deleted_at TIMESTAMPTZ;

-- 4. Replace UNIQUE constraints with partial unique indexes
ALTER TABLE customers DROP CONSTRAINT customers_name_key;
CREATE UNIQUE INDEX idx_customers_name_active ON customers(name) WHERE deleted_at IS NULL;

ALTER TABLE hardware_categories DROP CONSTRAINT hardware_categories_name_key;
CREATE UNIQUE INDEX idx_hardware_categories_name_active ON hardware_categories(name) WHERE deleted_at IS NULL;

ALTER TABLE user_assignments DROP CONSTRAINT user_assignments_user_id_customer_id_key;
CREATE UNIQUE INDEX idx_user_assignments_active ON user_assignments(user_id, customer_id) WHERE deleted_at IS NULL;

ALTER TABLE customer_services DROP CONSTRAINT customer_services_customer_id_service_id_key;
CREATE UNIQUE INDEX idx_customer_services_active ON customer_services(customer_id, service_id) WHERE deleted_at IS NULL;

ALTER TABLE category_field_definitions DROP CONSTRAINT category_field_definitions_category_id_name_key;
CREATE UNIQUE INDEX idx_field_definitions_name_active ON category_field_definitions(category_id, name) WHERE deleted_at IS NULL;

-- 5. Soft-delete guard triggers

-- customers: checks user_assignments, assets, licenses, customer_services
-- +goose StatementBegin
CREATE FUNCTION check_soft_delete_customers() RETURNS TRIGGER AS $$
BEGIN
    IF OLD.deleted_at IS NULL AND NEW.deleted_at IS NOT NULL THEN
        IF EXISTS (SELECT 1 FROM user_assignments WHERE customer_id = OLD.id AND deleted_at IS NULL) THEN
            RAISE EXCEPTION 'cannot delete: active dependent records exist' USING ERRCODE = '23503';
        END IF;
        IF EXISTS (SELECT 1 FROM assets WHERE customer_id = OLD.id AND deleted_at IS NULL) THEN
            RAISE EXCEPTION 'cannot delete: active dependent records exist' USING ERRCODE = '23503';
        END IF;
        IF EXISTS (SELECT 1 FROM licenses WHERE customer_id = OLD.id AND deleted_at IS NULL) THEN
            RAISE EXCEPTION 'cannot delete: active dependent records exist' USING ERRCODE = '23503';
        END IF;
        IF EXISTS (SELECT 1 FROM customer_services WHERE customer_id = OLD.id AND deleted_at IS NULL) THEN
            RAISE EXCEPTION 'cannot delete: active dependent records exist' USING ERRCODE = '23503';
        END IF;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER trg_customers_soft_delete_guard
    BEFORE UPDATE ON customers
    FOR EACH ROW EXECUTE FUNCTION check_soft_delete_customers();

-- users: checks user_assignments
-- +goose StatementBegin
CREATE FUNCTION check_soft_delete_users() RETURNS TRIGGER AS $$
BEGIN
    IF OLD.deleted_at IS NULL AND NEW.deleted_at IS NOT NULL THEN
        IF EXISTS (SELECT 1 FROM user_assignments WHERE user_id = OLD.id AND deleted_at IS NULL) THEN
            RAISE EXCEPTION 'cannot delete: active dependent records exist' USING ERRCODE = '23503';
        END IF;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER trg_users_soft_delete_guard
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION check_soft_delete_users();

-- services: checks customer_services
-- +goose StatementBegin
CREATE FUNCTION check_soft_delete_services() RETURNS TRIGGER AS $$
BEGIN
    IF OLD.deleted_at IS NULL AND NEW.deleted_at IS NOT NULL THEN
        IF EXISTS (SELECT 1 FROM customer_services WHERE service_id = OLD.id AND deleted_at IS NULL) THEN
            RAISE EXCEPTION 'cannot delete: active dependent records exist' USING ERRCODE = '23503';
        END IF;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER trg_services_soft_delete_guard
    BEFORE UPDATE ON services
    FOR EACH ROW EXECUTE FUNCTION check_soft_delete_services();

-- user_assignments: checks assets, licenses
-- +goose StatementBegin
CREATE FUNCTION check_soft_delete_user_assignments() RETURNS TRIGGER AS $$
BEGIN
    IF OLD.deleted_at IS NULL AND NEW.deleted_at IS NOT NULL THEN
        IF EXISTS (SELECT 1 FROM assets WHERE user_assignment_id = OLD.id AND deleted_at IS NULL) THEN
            RAISE EXCEPTION 'cannot delete: active dependent records exist' USING ERRCODE = '23503';
        END IF;
        IF EXISTS (SELECT 1 FROM licenses WHERE user_assignment_id = OLD.id AND deleted_at IS NULL) THEN
            RAISE EXCEPTION 'cannot delete: active dependent records exist' USING ERRCODE = '23503';
        END IF;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER trg_user_assignments_soft_delete_guard
    BEFORE UPDATE ON user_assignments
    FOR EACH ROW EXECUTE FUNCTION check_soft_delete_user_assignments();

-- hardware_categories: checks category_field_definitions, assets
-- +goose StatementBegin
CREATE FUNCTION check_soft_delete_hardware_categories() RETURNS TRIGGER AS $$
BEGIN
    IF OLD.deleted_at IS NULL AND NEW.deleted_at IS NOT NULL THEN
        IF EXISTS (SELECT 1 FROM category_field_definitions WHERE category_id = OLD.id AND deleted_at IS NULL) THEN
            RAISE EXCEPTION 'cannot delete: active dependent records exist' USING ERRCODE = '23503';
        END IF;
        IF EXISTS (SELECT 1 FROM assets WHERE category_id = OLD.id AND deleted_at IS NULL) THEN
            RAISE EXCEPTION 'cannot delete: active dependent records exist' USING ERRCODE = '23503';
        END IF;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER trg_hardware_categories_soft_delete_guard
    BEFORE UPDATE ON hardware_categories
    FOR EACH ROW EXECUTE FUNCTION check_soft_delete_hardware_categories();

-- 6. Attach audit triggers to all 9 tables
CREATE TRIGGER trg_customers_audit AFTER INSERT OR UPDATE ON customers FOR EACH ROW EXECUTE FUNCTION audit_trigger();
CREATE TRIGGER trg_users_audit AFTER INSERT OR UPDATE ON users FOR EACH ROW EXECUTE FUNCTION audit_trigger();
CREATE TRIGGER trg_services_audit AFTER INSERT OR UPDATE ON services FOR EACH ROW EXECUTE FUNCTION audit_trigger();
CREATE TRIGGER trg_user_assignments_audit AFTER INSERT OR UPDATE ON user_assignments FOR EACH ROW EXECUTE FUNCTION audit_trigger();
CREATE TRIGGER trg_assets_audit AFTER INSERT OR UPDATE ON assets FOR EACH ROW EXECUTE FUNCTION audit_trigger();
CREATE TRIGGER trg_licenses_audit AFTER INSERT OR UPDATE ON licenses FOR EACH ROW EXECUTE FUNCTION audit_trigger();
CREATE TRIGGER trg_customer_services_audit AFTER INSERT OR UPDATE ON customer_services FOR EACH ROW EXECUTE FUNCTION audit_trigger();
CREATE TRIGGER trg_hardware_categories_audit AFTER INSERT OR UPDATE ON hardware_categories FOR EACH ROW EXECUTE FUNCTION audit_trigger();
CREATE TRIGGER trg_category_field_definitions_audit AFTER INSERT OR UPDATE ON category_field_definitions FOR EACH ROW EXECUTE FUNCTION audit_trigger();

-- +goose Down

-- 1. Drop audit triggers from all 9 tables
DROP TRIGGER IF EXISTS trg_category_field_definitions_audit ON category_field_definitions;
DROP TRIGGER IF EXISTS trg_hardware_categories_audit ON hardware_categories;
DROP TRIGGER IF EXISTS trg_customer_services_audit ON customer_services;
DROP TRIGGER IF EXISTS trg_licenses_audit ON licenses;
DROP TRIGGER IF EXISTS trg_assets_audit ON assets;
DROP TRIGGER IF EXISTS trg_user_assignments_audit ON user_assignments;
DROP TRIGGER IF EXISTS trg_services_audit ON services;
DROP TRIGGER IF EXISTS trg_users_audit ON users;
DROP TRIGGER IF EXISTS trg_customers_audit ON customers;

-- 2. Drop guard triggers and functions
DROP TRIGGER IF EXISTS trg_hardware_categories_soft_delete_guard ON hardware_categories;
DROP FUNCTION IF EXISTS check_soft_delete_hardware_categories();

DROP TRIGGER IF EXISTS trg_user_assignments_soft_delete_guard ON user_assignments;
DROP FUNCTION IF EXISTS check_soft_delete_user_assignments();

DROP TRIGGER IF EXISTS trg_services_soft_delete_guard ON services;
DROP FUNCTION IF EXISTS check_soft_delete_services();

DROP TRIGGER IF EXISTS trg_users_soft_delete_guard ON users;
DROP FUNCTION IF EXISTS check_soft_delete_users();

DROP TRIGGER IF EXISTS trg_customers_soft_delete_guard ON customers;
DROP FUNCTION IF EXISTS check_soft_delete_customers();

-- 3. Drop partial unique indexes, restore original UNIQUE constraints
DROP INDEX IF EXISTS idx_field_definitions_name_active;
ALTER TABLE category_field_definitions ADD CONSTRAINT category_field_definitions_category_id_name_key UNIQUE (category_id, name);

DROP INDEX IF EXISTS idx_customer_services_active;
ALTER TABLE customer_services ADD CONSTRAINT customer_services_customer_id_service_id_key UNIQUE (customer_id, service_id);

DROP INDEX IF EXISTS idx_user_assignments_active;
ALTER TABLE user_assignments ADD CONSTRAINT user_assignments_user_id_customer_id_key UNIQUE (user_id, customer_id);

DROP INDEX IF EXISTS idx_hardware_categories_name_active;
ALTER TABLE hardware_categories ADD CONSTRAINT hardware_categories_name_key UNIQUE (name);

DROP INDEX IF EXISTS idx_customers_name_active;
ALTER TABLE customers ADD CONSTRAINT customers_name_key UNIQUE (name);

-- 4. Drop deleted_at columns from all 9 tables
ALTER TABLE category_field_definitions DROP COLUMN deleted_at;
ALTER TABLE hardware_categories DROP COLUMN deleted_at;
ALTER TABLE customer_services DROP COLUMN deleted_at;
ALTER TABLE licenses DROP COLUMN deleted_at;
ALTER TABLE assets DROP COLUMN deleted_at;
ALTER TABLE user_assignments DROP COLUMN deleted_at;
ALTER TABLE services DROP COLUMN deleted_at;
ALTER TABLE users DROP COLUMN deleted_at;
ALTER TABLE customers DROP COLUMN deleted_at;

-- 5. Drop audit_trigger function
DROP FUNCTION IF EXISTS audit_trigger();

-- 6. Drop audit_log table
DROP TABLE IF EXISTS audit_log;
