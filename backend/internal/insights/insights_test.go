package insights_test

import (
	"context"
	"encoding/json"
	"fmt"
	c "magistral/backend/internal/constraints"
	"magistral/backend/internal/dto"
	"magistral/backend/internal/insights"
	"magistral/backend/internal/planning"
	"os"
	"reflect"
	"testing"
	"time"
)

func world(t *testing.T, times ...string) *c.World {
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
	base := d.Orders[0]
	d.Orders = []dto.Order{}
	d.Now = "2026-10-26T08:30:01+05:00"
	for i, at := range times {
		o := base
		o.OrderId = fmt.Sprintf("ORD-20261026-%04d", i+1)
		o.RegisteredAt = dto.DateTime("2026-10-26T" + at + "+05:00")
		d.Orders = append(d.Orders, o)
	}
	w, e := c.New(r, d, m)
	if e != nil {
		t.Fatal(e)
	}
	return w
}
func detect(w *c.World) []dto.PgpGroup {
	return insights.Detect(w, nil, map[string]dto.PgpGroup{}, nil, nil, nil)
}
func TestPGPBoundaries(t *testing.T) {
	for _, test := range []struct {
		name   string
		times  []string
		groups int
	}{{"two", []string{"07:30:00", "08:00:00"}, 0}, {"three", []string{"07:30:00", "08:00:00", "08:15:00"}, 1}, {"exact_hour", []string{"07:30:00", "08:00:00", "08:30:00"}, 1}, {"hour_plus_second", []string{"07:30:00", "08:00:00", "08:30:01"}, 0}, {"separate_incidents", []string{"05:00:00", "05:10:00", "05:20:00", "07:00:00", "07:10:00", "07:20:00"}, 2}} {
		t.Run(test.name, func(t *testing.T) {
			w := world(t, test.times...)
			groups := detect(w)
			if len(groups) != test.groups {
				t.Fatalf("expected %d groups got %+v", test.groups, groups)
			}
			for _, g := range groups {
				if g.DetectedAt != w.Day.Now {
					t.Fatal("detection was backdated")
				}
			}
		})
	}
}
func TestPGPExclusionsAndIdentity(t *testing.T) {
	w := world(t, "07:30:00", "08:00:00", "08:20:00")
	original := detect(w)
	if len(original) != 1 {
		t.Fatal(original)
	}
	reversed := w.Day.Orders
	reversed[0], reversed[2] = reversed[2], reversed[0]
	if !reflect.DeepEqual(original, detect(w)) {
		t.Fatal("input order changes group identity")
	}
	history := map[string]dto.PgpGroup{original[0].GroupId: original[0]}
	w.Day.Now = "2026-10-26T08:40:00+05:00"
	cancel := map[string]dto.CancelledOrder{"ORD-20261026-0001": {OrderId: "ORD-20261026-0001"}}
	kept := insights.Detect(w, original, history, cancel, nil, nil)
	if len(kept) != 1 || len(kept[0].OrderIds) != 2 || kept[0].DetectedAt != original[0].DetectedAt || kept[0].GroupId != original[0].GroupId {
		t.Fatal("confirmed group lost its identity")
	}
	cancel["ORD-20261026-0002"] = dto.CancelledOrder{OrderId: "ORD-20261026-0002"}
	if g := insights.Detect(w, kept, history, cancel, nil, nil); len(g) != 0 {
		t.Fatal("singleton group violates contract")
	}
	w = world(t, "07:30:00", "08:00:00", "08:20:00")
	id := w.Day.Orders[0].OrderId
	for _, kind := range []string{"node", "irrelevant", "started", "cancelled", "transferred"} {
		t.Run(kind, func(t *testing.T) {
			x := world(t, "07:30:00", "08:00:00", "08:20:00")
			var cancelled map[string]dto.CancelledOrder
			var transferred map[string]bool
			switch kind {
			case "node":
				x.Day.Orders[0].AccessNodeId = "AN-0032"
			case "irrelevant":
				o := x.Day.Orders[0]
				o.WorkTypeCode = "INST_PON_MKD"
				x.Day.Orders[0] = o
			case "started":
				x.Previous = &dto.Plan{Routes: []dto.Route{{Stops: []dto.Stop{{Type: "VISIT", OrderId: &id, Start: "2026-10-26T08:00:00+05:00"}}}}}
			case "cancelled":
				cancelled = map[string]dto.CancelledOrder{id: {OrderId: id}}
			case "transferred":
				transferred = map[string]bool{id: true}
			}
			if g := insights.Detect(x, nil, nil, cancelled, transferred, nil); len(g) != 0 {
				t.Fatal("excluded order caused PGP", g)
			}
		})
	}
}
func TestGroupAccountingAndAlertAudit(t *testing.T) {
	w := world(t, "07:30:00", "08:00:00", "08:20:00")
	w.Groups = detect(w)
	p, e := planning.Build(context.Background(), w)
	if e != nil {
		t.Fatal(e)
	}
	fresh := insights.Alerts(w, nil, nil, &p)
	if len(p.Unassigned) != 0 || *p.Summary.VisitsPlanned != 0 || *p.Summary.PgpOrders != 3 || len(fresh) != 1 {
		t.Fatal("grouped orders also scheduled or lost")
	}
	if e = w.Validate(p); e != nil {
		t.Fatal(e)
	}
	old := p
	w.Previous = &old
	w.Day.Now = "2026-10-26T08:45:00+05:00"
	p.AsOf = w.Day.Now
	if a := insights.Alerts(w, old.Alerts, nil, &p); len(a) != 0 {
		t.Fatal("duplicate PGP alert")
	}
	if p.Alerts[0].CreatedAt != old.Alerts[0].CreatedAt {
		t.Fatal("creation time changed")
	}
	p.Alerts = nil
	if e = w.Validate(p); e == nil {
		t.Fatal("validator accepted deleted alert history")
	}
}
func TestAlertEpisodes(t *testing.T) {
	w := world(t, "08:00:00")
	order := w.Day.Orders[0]
	forecast := dto.DateTime("2026-10-26T11:30:00+05:00")
	risk := []insights.Risk{{OrderID: order.OrderId, StaffID: "ST-001", Forecast: &forecast}}
	p := dto.Plan{Unassigned: []dto.UnassignedOrder{{OrderId: order.OrderId, ReasonText: "Нужен перенос"}}}
	insights.Alerts(w, nil, risk, &p)
	if len(p.Alerts) != 2 || p.Alerts[0].Status != "ACTIVE" || *p.Alerts[0].LatenessMin != 30 {
		t.Fatal("missing late/unassigned alerts")
	}
	first := p.Alerts[0]
	w.Day.Now = "2026-10-26T08:40:00+05:00"
	later := dto.DateTime("2026-10-26T12:00:00+05:00")
	risk[0].Forecast = &later
	old := append([]dto.Alert{}, p.Alerts...)
	insights.Alerts(w, old, risk, &p)
	if p.Alerts[0].CreatedAt != first.CreatedAt || *p.Alerts[0].ForecastStart != forecast {
		t.Fatal("first risk evidence overwritten")
	}
	p.Unassigned = nil
	p.Routes = []dto.Route{{StaffId: "ST-029", Stops: []dto.Stop{{Type: "VISIT", OrderId: &order.OrderId}}}}
	insights.Alerts(w, p.Alerts, nil, &p)
	closed := p.Alerts[0]
	if closed.Status != "RESOLVED" || !*closed.Applied {
		t.Fatal("risk not resolved")
	}
	w.Day.Now = c.Stamp(c.Time(w.Day.Now).Add(time.Minute))
	p.Routes = nil
	p.Unassigned = []dto.UnassignedOrder{{OrderId: order.OrderId}}
	insights.Alerts(w, p.Alerts, risk, &p)
	if !reflect.DeepEqual(closed, p.Alerts[0]) || len(p.Alerts) != 4 || p.Alerts[2].AlertId == first.AlertId {
		t.Fatal("recurrence changed closed episode")
	}
}

func TestWarningThirtyMinuteBoundary(t *testing.T) {
	for _, at := range []string{"08:30:00", "08:30:01"} {
		t.Run(at, func(t *testing.T) {
			w := world(t, "08:00:00")
			w.Day.Now = dto.DateTime("2026-10-26T" + at + "+05:00")
			o := w.Day.Orders[0]
			forecast := dto.DateTime("2026-10-26T11:30:00+05:00")
			p := dto.Plan{Unassigned: []dto.UnassignedOrder{{OrderId: o.OrderId}}}
			insights.Alerts(w, nil, []insights.Risk{{OrderID: o.OrderId, StaffID: "ST-001", Forecast: &forecast}}, &p)
			a := p.Alerts[0]
			timely := !c.Time(a.CreatedAt).After(c.Time(o.Slot.Start).Add(-30 * time.Minute))
			if a.CreatedAt != w.Day.Now || timely != (at == "08:30:00") {
				t.Fatal("warning was backdated or 30-minute boundary changed")
			}
			if a.ProposedSlot == nil || a.Applied == nil || *a.Applied {
				t.Fatal("reschedule proposal missing or falsely applied")
			}
			if w.Day.Orders[0].Slot != o.Slot {
				t.Fatal("agreed slot was silently modified")
			}
		})
	}
}
