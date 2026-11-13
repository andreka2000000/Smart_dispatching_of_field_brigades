package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"magistral/backend/internal/application"
	"magistral/backend/internal/contract"
	"magistral/backend/internal/dto"
	"magistral/backend/internal/storage/postgres"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// Run against a real PostgreSQL; every test uses its own schema.
func TestStage1Postgres(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set TEST_DATABASE_URL for real PostgreSQL integration test")
	}
	ctx := context.Background()
	root, e := postgres.Open(ctx, url)
	if e != nil {
		t.Fatal(e)
	}
	defer root.Pool.Close()
	schema := fmt.Sprintf("stage1_test_%d", time.Now().UnixNano())
	if _, e = root.Pool.Exec(ctx, "CREATE SCHEMA "+schema); e != nil {
		t.Fatal(e)
	}
	defer root.Pool.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE")
	sep := "?"
	if strings.Contains(url, "?") {
		sep = "&"
	}
	testURL := url + sep + "search_path=" + schema
	store, e := postgres.Open(ctx, testURL)
	if e != nil {
		t.Fatal(e)
	}
	defer store.Pool.Close()
	v, e := contract.New()
	if e != nil {
		t.Fatal(e)
	}
	api := (&API{Service: &application.Service{Store: store}, Validator: v}).Handler()
	call := func(method, path string, body []byte, status int) []byte {
		t.Helper()
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(method, path, bytes.NewReader(body))
		api.ServeHTTP(rr, req)
		if rr.Code != status {
			t.Fatalf("%s %s: got %d want %d: %s", method, path, rr.Code, status, rr.Body.String())
		}
		if status >= 400 {
			var err map[string]any
			if json.Unmarshal(rr.Body.Bytes(), &err) != nil || err["error"] == nil {
				t.Fatal("nonstandard error")
			}
		}
		return rr.Body.Bytes()
	}
	read := func(name string) []byte {
		b, e := os.ReadFile("../../testdata/" + name + ".json")
		if e != nil {
			t.Fatal(e)
		}
		return b
	}
	ref, matrix, day := read("reference"), read("matrix"), read("day")
	dayPath := "/api/v1/days/2026-10-26"
	uiPath := "/ui/v1/days/2026-10-26"
	call("GET", "/api/v1/health", nil, 200)
	call("PUT", dayPath, day, 409)
	call("PUT", "/api/v1/reference/travel-matrix", matrix, 200) // matrix first is legal
	call("PUT", "/api/v1/reference", ref, 200)
	call("PUT", dayPath, day, 200)
	before := call("GET", uiPath, nil, 200)
	// Invalid links, duplicate IDs and malformed shapes must leave the day intact.
	call("PUT", dayPath, bytes.ReplaceAll(day, []byte("B-00412"), []byte("B-99999")), 422)
	call("PUT", "/api/v1/reference", bytes.ReplaceAll(ref, []byte("ST-029"), []byte("ST-001")), 422)
	call("PUT", "/api/v1/reference/travel-matrix", []byte(`{"matrix_id":"x","points":["a","b"],"durations_sec":[[0]],"distances_m":[[0]]}`), 422)
	call("PUT", dayPath, []byte(`{}`), 400)
	call("PUT", dayPath, []byte("null"), 400)
	call("PUT", dayPath, append(day, []byte(" {}")...), 400)
	if !bytes.Equal(before, call("GET", uiPath, nil, 200)) {
		t.Fatal("failed load changed state")
	}
	// Failure in SQL after deleting the existing day must roll back the deletion.
	_, e = store.Pool.Exec(ctx, `CREATE FUNCTION reject_order() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'injected failure'; END $$; CREATE TRIGGER reject_order BEFORE INSERT ON orders FOR EACH ROW EXECUTE FUNCTION reject_order()`)
	if e != nil {
		t.Fatal(e)
	}
	call("PUT", dayPath, day, 500)
	if !bytes.Equal(before, call("GET", uiPath, nil, 200)) {
		t.Fatal("SQL rollback failed")
	}
	if _, e = store.Pool.Exec(ctx, "DROP TRIGGER reject_order ON orders"); e != nil {
		t.Fatal(e)
	}
	call("PUT", dayPath, day, 200)
	after := call("GET", uiPath, nil, 200)
	if bytes.Equal(before, after) {
		t.Fatal("generation did not change")
	}
	// New process/store: now and data recover from PostgreSQL, not wall clock.
	recovered, e := postgres.Open(ctx, testURL)
	if e != nil {
		t.Fatal(e)
	}
	defer recovered.Pool.Close()
	api = (&API{Service: &application.Service{Store: recovered}, Validator: v}).Handler()
	if !bytes.Equal(after, call("GET", uiPath, nil, 200)) {
		t.Fatal("restart lost state")
	}
	call("GET", dayPath+"/plan", nil, 404)
	call("POST", "/api/v1/days/2099-01-01/plan", nil, 404)
	call("POST", dayPath+"/plan", []byte(`{"time_limit_sec":0}`), 400)
	planRaw := call("POST", dayPath+"/plan", []byte(`{"time_limit_sec":1}`), 200)
	var plan dto.Plan
	if e = v.Decode(planRaw, "Plan", &plan); e != nil {
		t.Fatal("plan violates OpenAPI", e)
	}
	if plan.PlanVersion != 1 || len(plan.Unassigned) != 0 || len(plan.Routes) != 2 || *plan.Summary.VisitsPlanned != 2 {
		t.Fatal("unexpected baseline")
	}
	if !bytes.Equal(planRaw, call("GET", dayPath+"/plan", nil, 200)) {
		t.Fatal("GET differs from committed POST")
	}
	// Read a committed plan from a freshly opened store.
	third, e := postgres.Open(ctx, testURL)
	if e != nil {
		t.Fatal(e)
	}
	defer third.Pool.Close()
	saved, e := third.CurrentPlan(ctx, "2026-10-26")
	if e != nil || saved.PlanVersion != 1 {
		t.Fatal("plan not persisted", e)
	}
	// Snapshot publication rejects both concurrent publication and reset input.
	inputs, e := store.PlanningInputs(ctx, "2026-10-26")
	if e != nil {
		t.Fatal(e)
	}
	if e = store.Publish(ctx, inputs, plan); e != nil {
		t.Fatal(e)
	}
	if e = store.Publish(ctx, inputs, plan); !errors.Is(e, postgres.ErrStale) {
		t.Fatal("concurrent plan overwrite accepted", e)
	}
	inputs, e = store.PlanningInputs(ctx, "2026-10-26")
	if e != nil {
		t.Fatal(e)
	}
	call("PUT", dayPath, day, 200)
	if e = store.Publish(ctx, inputs, plan); !errors.Is(e, postgres.ErrStale) {
		t.Fatal("stale generation accepted", e)
	}
	call("GET", dayPath+"/plan", nil, 404)
	call("POST", dayPath+"/plan", nil, 200)
	committed := call("GET", dayPath+"/plan", nil, 200)
	_, e = store.Pool.Exec(ctx, `CREATE FUNCTION reject_plan() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'injected snapshot failure'; END $$; CREATE TRIGGER reject_plan BEFORE INSERT ON plan_snapshots FOR EACH ROW EXECUTE FUNCTION reject_plan()`)
	if e != nil {
		t.Fatal(e)
	}
	call("POST", dayPath+"/plan", nil, 500)
	if !bytes.Equal(committed, call("GET", dayPath+"/plan", nil, 200)) {
		t.Fatal("snapshot failure changed current plan")
	}
	if _, e = store.Pool.Exec(ctx, "DROP TRIGGER reject_plan ON plan_snapshots; UPDATE days SET event_count=1"); e != nil {
		t.Fatal(e)
	}
	call("POST", dayPath+"/plan", nil, 409)
	if !bytes.Equal(committed, call("GET", dayPath+"/plan", nil, 200)) {
		t.Fatal("event guard changed plan")
	}
	call("PUT", dayPath, day, 200)
	call("POST", dayPath+"/plan", nil, 200)
	call("PUT", "/api/v1/reference/travel-matrix", matrix, 200)
	call("GET", dayPath+"/plan", nil, 404)
	call("POST", dayPath+"/plan", nil, 200)
	call("PUT", "/api/v1/reference", ref, 200)
	call("GET", uiPath, nil, 404)
	call("GET", dayPath+"/plan", nil, 404)
	call("PUT", dayPath, day, 200) // matrix survived reference replacement
}
