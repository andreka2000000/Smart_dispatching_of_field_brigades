CREATE TABLE IF NOT EXISTS schema_migrations (version integer PRIMARY KEY);
CREATE TABLE IF NOT EXISTS reference_data (id boolean PRIMARY KEY DEFAULT true CHECK(id), version bigint NOT NULL, document jsonb NOT NULL);
CREATE TABLE IF NOT EXISTS travel_matrices (id boolean PRIMARY KEY DEFAULT true CHECK(id), version bigint NOT NULL, document jsonb NOT NULL);
CREATE TABLE IF NOT EXISTS service_centers (id text PRIMARY KEY, document jsonb NOT NULL);
CREATE TABLE IF NOT EXISTS staff (id text PRIMARY KEY, service_center_id text NOT NULL REFERENCES service_centers(id), document jsonb NOT NULL);
CREATE TABLE IF NOT EXISTS access_nodes (id text PRIMARY KEY, document jsonb NOT NULL);
CREATE TABLE IF NOT EXISTS buildings (id text PRIMARY KEY, document jsonb NOT NULL);
CREATE TABLE IF NOT EXISTS work_types (id text PRIMARY KEY, document jsonb NOT NULL);
CREATE SEQUENCE IF NOT EXISTS day_generation;
CREATE TABLE IF NOT EXISTS days (date date PRIMARY KEY, generation bigint NOT NULL DEFAULT nextval('day_generation'), model_now timestamptz NOT NULL, document jsonb NOT NULL);
CREATE TABLE IF NOT EXISTS orders (day date NOT NULL REFERENCES days(date) ON DELETE CASCADE, id text NOT NULL, building_id text NOT NULL REFERENCES buildings(id), work_type_code text NOT NULL REFERENCES work_types(id), document jsonb NOT NULL, PRIMARY KEY(day,id));
INSERT INTO schema_migrations(version) VALUES(1) ON CONFLICT DO NOTHING;
ALTER TABLE days ADD COLUMN IF NOT EXISTS plan_revision bigint NOT NULL DEFAULT 0;
ALTER TABLE days ADD COLUMN IF NOT EXISTS current_version bigint NOT NULL DEFAULT 0;
ALTER TABLE days ADD COLUMN IF NOT EXISTS event_count bigint NOT NULL DEFAULT 0;
CREATE TABLE IF NOT EXISTS plan_snapshots (
 day date NOT NULL REFERENCES days(date) ON DELETE CASCADE,
 generation bigint NOT NULL,
 revision bigint NOT NULL,
 plan_version bigint NOT NULL,
 reference_version bigint NOT NULL,
 matrix_version bigint NOT NULL,
 document jsonb NOT NULL,
 PRIMARY KEY(day,generation,revision)
);
INSERT INTO schema_migrations(version) VALUES(2) ON CONFLICT DO NOTHING;
