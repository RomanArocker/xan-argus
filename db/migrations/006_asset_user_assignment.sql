-- +goose Up

ALTER TABLE assets
    ADD COLUMN user_assignment_id UUID REFERENCES user_assignments(id) ON DELETE RESTRICT;

-- Consistency trigger: ensure user_assignment belongs to same customer as the asset
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION check_asset_customer_consistency()
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
            RAISE EXCEPTION 'asset customer_id (%) does not match user_assignment customer_id (%)',
                NEW.customer_id, assignment_customer_id;
        END IF;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER trg_assets_customer_consistency
    BEFORE INSERT OR UPDATE ON assets
    FOR EACH ROW EXECUTE FUNCTION check_asset_customer_consistency();

-- +goose Down
DROP TRIGGER IF EXISTS trg_assets_customer_consistency ON assets;
DROP FUNCTION IF EXISTS check_asset_customer_consistency;
ALTER TABLE assets DROP COLUMN IF EXISTS user_assignment_id;
