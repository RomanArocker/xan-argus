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
