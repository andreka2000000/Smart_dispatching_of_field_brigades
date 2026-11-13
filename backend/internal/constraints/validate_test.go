package constraints_test

import (
	"context"
	"encoding/json"
	"errors"
	c "magistral/backend/internal/constraints"
	"magistral/backend/internal/dto"
	"magistral/backend/internal/planning"
	"os"
	"testing"
	"time"
)

func fixture(t *testing.T) (*c.World, dto.Plan) {
	t.Helper()
	var r dto.ReferenceData
	var d dto.DayInput
	var m dto.TravelMatrix
	for name, out := range map[string]any{"reference": &r, "day": &d, "matrix": &m} {
		b, e := os.ReadFile("../../testdata/" + name + ".json")
		if e != nil {
			t.Fatal(e)
		}
		if e = json.Unmarshal(b, out); e != nil {
			t.Fatal(e)
		}
	}
	w, e := c.New(r, d, m)
	if e != nil {
		t.Fatal(e)
	}
	p, e := planning.Build(context.Background(), w)
	if e != nil {
		t.Fatal(e)
	}
	return w, p
}
func visit(p *dto.Plan) *dto.Stop {
	for i := range p.Routes {
		for j := range p.Routes[i].Stops {
			if p.Routes[i].Stops[j].Type == "VISIT" {
				return &p.Routes[i].Stops[j]
			}
		}
	}
	panic("no visit")
}
func TestH1ThroughH10(t *testing.T) {
	tests := []struct {
		code   string
		change func(*c.World, *dto.Plan)
	}{
		{"H1", func(w *c.World, p *dto.Plan) {
			s := w.Staff[p.Routes[0].StaffId]
			s.Technologies = []dto.Technology{"DOCSIS"}
			w.Staff[s.StaffId] = s
		}},
		{"H2", func(w *c.World, p *dto.Plan) { p.Routes[0].Stops[0].Arrival = "2026-10-26T08:59:59+05:00" }},
		{"H3", func(w *c.World, p *dto.Plan) {
			for i := range p.Routes {
				r := &p.Routes[i]
				for j, x := range r.Stops {
					if x.Type == "BREAK" {
						r.Stops = append(r.Stops[:j], r.Stops[j+1:]...)
						for k := range r.Stops {
							r.Stops[k].Seq = int64(k + 1)
						}
						return
					}
				}
			}
		}},
		{"H4", func(w *c.World, p *dto.Plan) {
			v := visit(p)
			o := w.Orders[*v.OrderId]
			v.Start = c.Stamp(c.Time(o.LatestStart).Add(time.Second))
			v.End = c.Stamp(c.Time(v.Start).Add(time.Hour))
		}},
		{"H5", func(w *c.World, p *dto.Plan) { v := visit(p); v.End = c.Stamp(c.Time(v.End).Add(time.Second)) }},
		{"H6", func(w *c.World, p *dto.Plan) { v := visit(p); v.Arrival = c.Stamp(c.Time(v.Arrival).Add(-time.Second)) }},
		{"H7", func(w *c.World, p *dto.Plan) {
			s := w.Staff[p.Routes[0].StaffId]
			s.EquipmentStart = dto.Equipment{}
			w.Staff[s.StaffId] = s
		}},
		{"H8", func(w *c.World, p *dto.Plan) { p.Routes[0].ServiceCenterId = "SC-99" }},
		{"H9", func(w *c.World, p *dto.Plan) {
			p.Unassigned = append(p.Unassigned, dto.UnassignedOrder{OrderId: *visit(p).OrderId})
		}},
		{"H10", func(w *c.World, p *dto.Plan) {
			b, _ := json.Marshal(p)
			var old dto.Plan
			json.Unmarshal(b, &old)
			w.Previous = &old
			w.Day.Now = "2026-10-26T09:30:00+05:00"
			p.AsOf = w.Day.Now
			for j := 1; j < len(p.Routes[0].Stops); j++ {
				x := &p.Routes[0].Stops[j]
				x.Arrival = c.Stamp(c.Time(x.Arrival).Add(time.Second))
				x.Start = c.Stamp(c.Time(x.Start).Add(time.Second))
				x.End = c.Stamp(c.Time(x.End).Add(time.Second))
			}
		}},
	}
	for _, test := range tests {
		t.Run(test.code, func(t *testing.T) {
			w, p := fixture(t)
			test.change(w, &p)
			e := w.Validate(p)
			var v *c.Violation
			if !errors.As(e, &v) || v.Code != test.code {
				t.Fatalf("expected %s got %v", test.code, e)
			}
		})
	}
}
func TestBackdatedNewStop(t *testing.T) {
	w, p := fixture(t)
	old := p
	old.Routes = []dto.Route{}
	w.Previous = &old
	w.Day.Now = "2026-10-26T10:00:00+05:00"
	p.AsOf = w.Day.Now
	if e := w.Validate(p); e == nil {
		t.Fatal("backdated new stops accepted")
	}
}
func TestFrozenBoundary(t *testing.T) {
	w, p := fixture(t)
	old := p
	w.Previous = &old
	w.Day.Now = "2026-10-26T09:00:00+05:00"
	p.AsOf = w.Day.Now
	if e := w.Validate(p); e != nil {
		t.Fatal(e)
	}
	old.Routes[0].Stops = append([]dto.Stop(nil), p.Routes[0].Stops...)
	w.Previous = &old
	// A departure at exactly now is not frozen; one second earlier is frozen.
	w.Day.Now = "2026-10-26T09:00:01+05:00"
	p.AsOf = w.Day.Now
	p.Routes[0].Stops[0].Arrival = c.Stamp(c.Time(p.Routes[0].Stops[0].Arrival).Add(time.Second))
	p.Routes[0].Stops[0].Start = p.Routes[0].Stops[0].Arrival
	p.Routes[0].Stops[0].End = p.Routes[0].Stops[0].Arrival
	if e := w.Validate(p); e == nil {
		t.Fatal("frozen depot changed")
	}
}
