-- db/migrations/007_reset_required_fields.sql
-- Required field validation is deferred to post-MVP.
-- Reset all category_field_definitions to required = false.

-- +goose Up
UPDATE category_field_definitions SET required = false WHERE required = true;

-- +goose Down
-- No rollback — the previous values are not recoverable without seed data knowledge.
