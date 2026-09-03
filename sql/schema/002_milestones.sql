-- +goose Up
ALTER TABLE milestones
    ADD COLUMN description TEXT,
    ADD COLUMN date_precision VARCHAR(20) NOT NULL DEFAULT 'exact';

-- +goose Down
ALTER TABLE milestones
    DROP COLUMN description,
    DROP COLUMN date_precision;