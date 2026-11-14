package transport

import (
	"encoding/json"
	"magistral/backend/internal/application"
	"magistral/backend/internal/dto"
	"magistral/backend/internal/optimization"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestOptimizerMetadataPersistenceAndEventIsolation(t *testing.T) {
	f := eventFixture(t)
	var calls atomic.Int64
	worker := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		json.NewEncoder(w).Encode(optimization.Response{Version: "0.14.0", Candidates: []optimization.Candidate{{Model: "fixture", Routes: []optimization.Route{{StaffID: "ST-001", OrderIDs: []string{"ORD-20261026-0001"}}, {StaffID: "ST-029", OrderIDs: []string{"ORD-20261026-0042"}}}}}})
	}))
	defer worker.Close()
	client, e := optimization.New(worker.URL)
	if e != nil {
		t.Fatal(e)
	}
	f.api = (&API{Service: &application.Service{Store: f.store, Optimizer: client}, Validator: f.validator}).Handler()
	raw := f.call("POST", dayURL+"/plan", []byte(`{"time_limit_sec":10}`), 200)
	var created dto.Plan
	if e = json.Unmarshal(raw, &created); e != nil {
		t.Fatal(e)
	}
	stored := f.plan()
	if calls.Load() != 1 || created.Optimization == nil || stored.Optimization == nil || stored.Optimization.CandidatesChecked != 1 {
		t.Fatal("optimization report not committed with plan")
	}
	f.event(tick("after-optimization", "08:45:00"), 200)
	if f.plan().Optimization != nil {
		t.Fatal("event snapshot falsely claims initial optimization")
	}
	f.call("POST", dayURL+"/plan", []byte(`{"time_limit_sec":10}`), 409)
	if calls.Load() != 1 {
		t.Fatal("worker invoked after accepted events")
	}
}
