package transport

import (
	"context"
	"encoding/json"
	"magistral/backend/internal/application"
	"magistral/backend/internal/routing"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRoadRouteDoesNotMutatePlan(t *testing.T) {
	f := eventFixture(t)
	path := "/ui/v1/days/2026-10-26/staff/ST-001/road-route"
	f.call("GET", path, nil, 409)
	f.build()
	f.call("GET", path, nil, 503)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		coords := []routing.Point{{60.6105, 56.8941}, {60.6531, 56.8712}, {60.6105, 56.8941}}
		legs := []any{}
		for i := 1; i < 3; i++ {
			legs = append(legs, map[string]any{"duration": 100, "distance": 500, "steps": []any{map[string]any{"geometry": routing.Geometry{Type: "LineString", Coordinates: coords[i-1 : i+1]}}}})
		}
		json.NewEncoder(w).Encode(map[string]any{"code": "Ok", "routes": []any{map[string]any{"geometry": routing.Geometry{Type: "LineString", Coordinates: coords}, "legs": legs}}})
	}))
	defer server.Close()
	client, _ := routing.New(server.URL)
	f.api = (&API{Service: &application.Service{Store: f.store}, Validator: f.validator, Router: client}).Handler()
	var response struct {
		routing.Path
		Version int64  `json:"plan_version"`
		Traffic string `json:"traffic"`
	}
	raw := f.call("GET", path+"?version=1", nil, 200)
	if e := json.Unmarshal(raw, &response); e != nil {
		t.Fatal(e)
	}
	if response.Version != 1 || response.Traffic != "unavailable" || len(response.Legs) != 2 || response.Legs[1].FromSeq != 3 || response.Legs[1].ToSeq != 4 {
		t.Fatalf("invalid leg mapping: %s", raw)
	}
	f.call("GET", path+"?version=99", nil, 409)
	f.call("GET", "/ui/v1/days/2026-10-26/staff/missing/road-route", nil, 404)
	snapshot, e := f.store.Workspace(context.Background(), "2026-10-26")
	if e != nil || snapshot.Plan.PlanVersion != 1 || snapshot.EventCount != 0 {
		t.Fatal("road display changed the model plan", e)
	}
}
