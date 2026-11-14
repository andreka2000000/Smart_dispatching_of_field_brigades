package optimization

import (
	"context"
	"encoding/json"
	c "magistral/backend/internal/constraints"
	"magistral/backend/internal/dto"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func world(t *testing.T) *c.World {
	t.Helper()
	var r dto.ReferenceData
	var d dto.DayInput
	var m dto.TravelMatrix
	for name, out := range map[string]any{"reference": &r, "day": &d, "matrix": &m} {
		raw, e := os.ReadFile("../../testdata/" + name + ".json")
		if e != nil {
			t.Fatal(e)
		}
		if e = json.Unmarshal(raw, out); e != nil {
			t.Fatal(e)
		}
	}
	// Greedy trap: flexible ST-001 takes the PON job, leaving no specialist for
	// overlapping FTTB work. The proposal reserves ST-001 for FTTB instead.
	first := r.Staff[0]
	second := first
	second.StaffId = "ST-002"
	second.Technologies = []dto.Technology{"PON"}
	r.Staff = []dto.Staff{first, second}
	a := d.Orders[0]
	a.Slot.End = "2026-10-26T09:30:00+05:00"
	a.LatestStart = a.Slot.End
	b := a
	b.OrderId = "ORD-20261026-0002"
	b.Technology = "FTTB"
	b.Slot.Start = "2026-10-26T09:10:00+05:00"
	b.Slot.End = "2026-10-26T09:40:00+05:00"
	b.LatestStart = b.Slot.End
	d.Orders = []dto.Order{a, b}
	w, e := c.New(r, d, m)
	if e != nil {
		t.Fatal(e)
	}
	return w
}
func TestChooseImprovedProposal(t *testing.T) {
	w := world(t)
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		var request map[string]json.RawMessage
		if e := json.NewDecoder(r.Body).Decode(&request); e != nil {
			t.Error(e)
		}
		if request["matrix"] == nil || request["excluded_order_ids"] == nil {
			t.Error("missing authoritative inputs")
		}
		json.NewEncoder(rw).Encode(Response{Version: "0.14.0", Candidates: []Candidate{{Model: "fixture", Routes: []Route{{StaffID: "ST-002", OrderIDs: []string{w.Day.Orders[0].OrderId}}, {StaffID: "ST-001", OrderIDs: []string{w.Day.Orders[1].OrderId}}}, ProposedVisits: 2}}})
	}))
	defer srv.Close()
	client, _ := New(srv.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	p, e := Build(ctx, w, client)
	if e != nil {
		t.Fatal(e)
	}
	if e = w.Validate(p); e != nil {
		t.Fatal(e)
	}
	if p.Optimization.Engine != "pyvrp_go" || p.Optimization.Baseline.Visits != 1 || p.Optimization.Selected.Visits != 2 {
		t.Fatalf("improvement not selected: %+v", p.Optimization)
	}
}
func TestUntrustedOrUnavailableWorkerFallsBack(t *testing.T) {
	for _, test := range []string{"unavailable", "malformed", "invalid proposal", "inferior", "timeout", "oversize"} {
		t.Run(test, func(t *testing.T) {
			w := world(t)
			srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
				switch test {
				case "unavailable":
					rw.WriteHeader(503)
				case "malformed":
					rw.Write([]byte("bad json"))
				case "invalid proposal":
					json.NewEncoder(rw).Encode(Response{Version: "0.14.0", Candidates: []Candidate{{Routes: []Route{{StaffID: "ST-002", OrderIDs: []string{w.Day.Orders[1].OrderId}}}}}})
				case "inferior":
					json.NewEncoder(rw).Encode(Response{Version: "0.14.0", Candidates: []Candidate{{Routes: []Route{}}}})
				case "timeout":
					select {
					case <-r.Context().Done():
					case <-time.After(100 * time.Millisecond):
					}
				case "oversize":
					rw.Write(make([]byte, (1<<20)+1))
				}
			}))
			defer srv.Close()
			client, _ := New(srv.URL)
			if test == "timeout" {
				client.http.Timeout = 30 * time.Millisecond
			}
			ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
			defer cancel()
			p, e := Build(ctx, w, client)
			if e != nil || w.Validate(p) != nil {
				t.Fatal("fallback invalid", e)
			}
			if p.Optimization.Engine != "go_baseline" || p.Optimization.Selected.Visits != 1 {
				t.Fatal("baseline lost", p.Optimization)
			}
		})
	}
}
func TestShortBudgetSkipsWorkerAndHistoryRejected(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Error("worker called with short budget") }))
	defer srv.Close()
	client, _ := New(srv.URL)
	w := world(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	p, e := Build(ctx, w, client)
	if e != nil || p.Optimization.Status != "short_budget" {
		t.Fatal(e)
	}
	w.Previous = &p
	if _, e = Build(ctx, w, client); e == nil {
		t.Fatal("initial solver may not replan history")
	}
}
