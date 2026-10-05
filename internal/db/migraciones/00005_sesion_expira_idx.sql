-- +goose Up
CREATE INDEX sesion_expira_idx ON sesion (expira_en);

-- +goose Down
DROP INDEX sesion_expira_idx;
