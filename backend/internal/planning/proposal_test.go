package planning

import (
	"context"
	"magistral/backend/internal/dto"
	"testing"
)

func TestProposalTrustBoundary(t *testing.T) {
	for _, test := range []string{"unknown staff", "absent staff", "duplicate job", "unknown job", "qualification", "PGP", "history"} {
		t.Run(test, func(t *testing.T) {
			w := load(t)
			id := w.Day.Orders[0].OrderId
			seed := map[string][]string{"ST-001": {id}}
			switch test {
			case "unknown staff":
				seed = map[string][]string{"unknown": {id}}
			case "absent staff":
				w.Day.AbsentStaffIds = []string{"ST-001"}
			case "duplicate job":
				seed["ST-001"] = []string{id, id}
			case "unknown job":
				seed["ST-001"] = []string{"unknown"}
			case "qualification":
				seed = map[string][]string{"ST-001": {w.Day.Orders[1].OrderId}}
			case "PGP":
				w.Groups = []dto.PgpGroup{{OrderIds: []string{id}}}
			case "history":
				w.Previous = &dto.Plan{}
			}
			if _, e := FromSequences(context.Background(), w, seed); e == nil {
				t.Fatal("untrusted proposal accepted")
			}
		})
	}
}
func TestProposalRebuildsEquipmentAndLunch(t *testing.T) {
	w := load(t)
	s := w.Staff["ST-001"]
	s.EquipmentStart = dto.Equipment{}
	w.Staff[s.StaffId] = s
	w.Reference.Staff[0] = s
	p, e := FromSequences(context.Background(), w, map[string][]string{s.StaffId: {w.Day.Orders[0].OrderId}})
	if e != nil {
		t.Fatal(e)
	}
	if e = w.Validate(p); e != nil {
		t.Fatal(e)
	}
	var restock, lunch bool
	for _, r := range p.Routes {
		if r.StaffId == s.StaffId {
			for _, stop := range r.Stops {
				restock = restock || stop.Type == "RESTOCK"
				lunch = lunch || stop.Type == "BREAK"
			}
		}
	}
	if !restock || !lunch || Quality(w, p).Visits != 2 {
		t.Fatal("proposal did not receive exact restock/lunch and insertion repair")
	}
}
func TestQualityPriority(t *testing.T) {
	base := dto.PlanQuality{WeightedVisits: 10, Visits: 8, TravelSec: 100, UtilizationCV: .2}
	for _, q := range []dto.PlanQuality{{WeightedVisits: 11, Visits: 7, TravelSec: 999}, {WeightedVisits: 10, Visits: 9, TravelSec: 999}, {WeightedVisits: 10, Visits: 8, TravelSec: 99}, {WeightedVisits: 10, Visits: 8, TravelSec: 100, UtilizationCV: .1}} {
		if !Better(q, base) || Better(base, q) {
			t.Fatal("quality ordering broken", q)
		}
	}
	if Better(base, base) {
		t.Fatal("equal plan must not replace baseline")
	}
}
