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
ALTER TABLE days ADD COLUMN IF NOT EXISTS event_state jsonb NOT NULL DEFAULT '{}';
CREATE TABLE IF NOT EXISTS events (
 day date NOT NULL REFERENCES days(date) ON DELETE CASCADE,
 generation bigint NOT NULL,
 event_id text NOT NULL,
 timestamp timestamptz NOT NULL,
 type text NOT NULL,
 plan_version bigint NOT NULL,
 document jsonb NOT NULL,
 result jsonb NOT NULL,
 PRIMARY KEY(day,generation,event_id),
 UNIQUE(day,generation,plan_version)
);
CREATE INDEX IF NOT EXISTS events_chronology ON events(day,generation,plan_version);
INSERT INTO schema_migrations(version) VALUES(3) ON CONFLICT DO NOTHING;
-- Application identity is independent from replaceable case reference data.
CREATE TABLE IF NOT EXISTS app_users (
 id text PRIMARY KEY, username text NOT NULL UNIQUE, name text NOT NULL,
 role text NOT NULL CHECK(role IN ('DISPATCHER','TECHNICIAN','SUBSCRIBER')),
 staff_id text, password_hash text NOT NULL, disabled boolean NOT NULL DEFAULT false,
 created_at timestamptz NOT NULL DEFAULT now(),
 CHECK ((role='TECHNICIAN' AND staff_id IS NOT NULL) OR (role<>'TECHNICIAN' AND staff_id IS NULL))
);
CREATE UNIQUE INDEX IF NOT EXISTS app_users_staff ON app_users(staff_id) WHERE role='TECHNICIAN' AND NOT disabled;
CREATE TABLE IF NOT EXISTS app_sessions (
 token_hash bytea PRIMARY KEY, user_id text NOT NULL REFERENCES app_users(id) ON DELETE CASCADE,
 expires_at timestamptz NOT NULL, created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS app_sessions_user ON app_sessions(user_id);
CREATE TABLE IF NOT EXISTS app_login_limits (key text PRIMARY KEY, attempts integer NOT NULL, reset_at timestamptz NOT NULL);
CREATE TABLE IF NOT EXISTS service_requests (
 id text PRIMARY KEY, user_id text NOT NULL REFERENCES app_users(id),
 service text NOT NULL CHECK(service IN ('INTERNET','REPAIR','TV')),
 address text NOT NULL, phone text NOT NULL, description text NOT NULL,
 preferred_date text NOT NULL DEFAULT '', status text NOT NULL DEFAULT 'NEW' CHECK(status IN ('NEW','IN_REVIEW','CLOSED')),
 decision_note text NOT NULL DEFAULT '', created_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS service_requests_owner ON service_requests(user_id,created_at DESC);
INSERT INTO schema_migrations(version) VALUES(4) ON CONFLICT DO NOTHING;
ALTER TABLE service_requests DROP CONSTRAINT IF EXISTS service_requests_status_check;
ALTER TABLE service_requests ADD CONSTRAINT service_requests_status_check CHECK(status IN ('NEW','IN_REVIEW','APPROVED','REJECTED','CLOSED'));
CREATE TABLE IF NOT EXISTS request_orders (
 request_id text PRIMARY KEY REFERENCES service_requests(id), day text NOT NULL,generation bigint NOT NULL,
 order_id text NOT NULL,result jsonb NOT NULL, UNIQUE(day,generation,order_id)
);
CREATE TABLE IF NOT EXISTS visit_facts (
 day text NOT NULL,generation bigint NOT NULL,order_id text NOT NULL,staff_id text NOT NULL,
 status text NOT NULL CHECK(status IN ('IN_PROGRESS','COMPLETED')),started_at timestamptz NOT NULL,
 completed_at timestamptz,report text NOT NULL DEFAULT '', PRIMARY KEY(day,generation,order_id)
);
INSERT INTO schema_migrations(version) VALUES(5) ON CONFLICT DO NOTHING;
ALTER TABLE service_requests ADD COLUMN IF NOT EXISTS client_key text;
ALTER TABLE service_requests ADD COLUMN IF NOT EXISTS payload_hash text;
CREATE UNIQUE INDEX IF NOT EXISTS service_requests_client_key ON service_requests(user_id,client_key) WHERE client_key IS NOT NULL;
INSERT INTO schema_migrations(version) VALUES(6) ON CONFLICT DO NOTHING;
