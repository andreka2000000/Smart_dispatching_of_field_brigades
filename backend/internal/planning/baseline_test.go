package planning

import (
	"context"
	"encoding/json"
	"fmt"
	c "magistral/backend/internal/constraints"
	"magistral/backend/internal/dto"
	"os"
	"testing"
	"time"
)

func load(t *testing.T) *c.World {
	t.Helper()
	var r dto.ReferenceData
	var d dto.DayInput
	var m dto.TravelMatrix
	for n, out := range map[string]any{"reference": &r, "day": &d, "matrix": &m} {
		b, e := os.ReadFile("../../testdata/" + n + ".json")
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
	return w
}

func TestScaleBudget(t *testing.T) {
	base := load(t)
	r := base.Reference
	d := base.Day
	r.Staff = []dto.Staff{}
	d.Orders = []dto.Order{}
	s := base.Reference.Staff[0]
	s.Profiles = []dto.Profile{"INSTALL", "LTP3"}
	s.EquipmentStart = dto.Equipment{ONT: 50, ROUTER: 50, STB: 50}
	s.EquipmentCapacity = s.EquipmentStart
	for i := 1; i <= 40; i++ {
		s.StaffId = fmt.Sprintf("ST-%03d", i)
		r.Staff = append(r.Staff, s)
	}
	for i := 1; i <= 400; i++ {
		o := base.Day.Orders[0]
		o.OrderId = fmt.Sprintf("ORD-20261026-%04d", i)
		o.Slot.End = "2026-10-26T17:00:00+05:00"
		o.LatestStart = o.Slot.End
		o.Deadline = "2026-10-26T18:00:00+05:00"
		d.Orders = append(d.Orders, o)
	}
	w, e := c.New(r, d, base.Matrix.Data)
	if e != nil {
		t.Fatal(e)
	}
	started := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	p, e := Build(ctx, w)
	if e != nil {
		t.Fatal(e)
	}
	if time.Since(started) > 3*time.Second {
		t.Fatal("budget not respected")
	}
	if len(p.Routes) != 40 || *p.Summary.VisitsPlanned < 1 || *p.Summary.VisitsPlanned+int64(len(p.Unassigned)) != 400 {
		t.Fatal("incomplete result")
	}
	if e = w.Validate(p); e != nil {
		t.Fatal(e)
	}
	t.Logf("40 staff / 400 orders: %d visits in %v", *p.Summary.VisitsPlanned, time.Since(started))
}
func TestBaselineAndCancelledBudget(t *testing.T) {
	w := load(t)
	p, e := Build(context.Background(), w)
	if e != nil {
		t.Fatal(e)
	}
	if len(p.Unassigned) != 0 || *p.Summary.VisitsPlanned != 2 {
		t.Fatalf("demo visits: %+v", p.Summary)
	}
	a, _ := json.Marshal(p)
	p2, e := Build(context.Background(), w)
	if e != nil {
		t.Fatal(e)
	}
	b, _ := json.Marshal(p2)
	if string(a) != string(b) {
		t.Fatal("nondeterministic baseline")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	p, e = Build(ctx, w)
	if e != nil || len(p.Unassigned) != 2 || len(p.Routes) != 2 {
		t.Fatal("deadline fallback invalid", e)
	}
	if w.Validate(p) != nil {
		t.Fatal("fallback fails validator")
	}
}
func TestRestockAndBreak(t *testing.T) {
	w := load(t)
	s := w.Staff["ST-001"]
	s.EquipmentStart = dto.Equipment{}
	w.Staff[s.StaffId] = s
	for i := range w.Reference.Staff {
		if w.Reference.Staff[i].StaffId == s.StaffId {
			w.Reference.Staff[i] = s
		}
	}
	p, e := Build(context.Background(), w)
	if e != nil {
		t.Fatal(e)
	}
	restocks := 0
	for _, r := range p.Routes {
		for _, x := range r.Stops {
			if x.Type == "RESTOCK" {
				restocks++
			}
		}
	}
	if restocks == 0 || len(p.Unassigned) > 0 {
		t.Fatal("restock not planned")
	}
	center := w.Reference.ServiceCenters[0]
	center.OpeningHours.Open = "19:00"
	center.OpeningHours.Close = "21:00"
	w.Reference.ServiceCenters[0] = center
	w.Centers[center.ServiceCenterId] = center
	p, e = Build(context.Background(), w)
	if e != nil {
		t.Fatal(e)
	}
	if len(p.Unassigned) != 1 {
		t.Fatal("closed restock center should leave REPAIR unassigned")
	}
}
func TestAbsentAndEquipmentDiagnostics(t *testing.T) {
	w := load(t)
	w.Day.AbsentStaffIds = []string{"ST-001"}
	p, e := Build(context.Background(), w)
	if e != nil {
		t.Fatal(e)
	}
	if len(p.Routes) != 1 || len(p.Unassigned) != 1 || p.Unassigned[0].ReasonCode != "NO_QUALIFIED_STAFF" {
		t.Fatal("absent staff used")
	}
	w = load(t)
	o := w.Day.Orders[0]
	o.RequiredEquipment.ONT = 99
	w.Day.Orders[0] = o
	w.Orders[o.OrderId] = o
	p, e = Build(context.Background(), w)
	if e != nil {
		t.Fatal(e)
	}
	if len(p.Unassigned) != 1 || p.Unassigned[0].ReasonCode != "NO_EQUIPMENT" {
		t.Fatal("capacity diagnostic")
	}
}
