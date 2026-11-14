package planning

import (
	"context"
	"fmt"
	c "magistral/backend/internal/constraints"
	"magistral/backend/internal/dto"
)

// FromSequences rebuilds a solver proposal using the exact Go simulator. No
// solver timestamps or inventory values are trusted. It is initial-plan only.
func FromSequences(ctx context.Context, w *c.World, seed map[string][]string) (dto.Plan, error) {
	if w.Previous != nil {
		return dto.Plan{}, fmt.Errorf("optimizer proposals are not accepted for event history")
	}
	excluded := map[string]bool{}
	for _, g := range w.Groups {
		for _, id := range g.OrderIds {
			excluded[id] = true
		}
	}
	seen := map[string]bool{}
	for staff, ids := range seed {
		person, ok := w.Staff[staff]
		if !ok || c.Has(w.Day.AbsentStaffIds, staff) {
			return dto.Plan{}, fmt.Errorf("proposal references unknown/absent staff")
		}
		for _, id := range ids {
			order, ok := w.Orders[id]
			if !ok || seen[id] || excluded[id] {
				return dto.Plan{}, fmt.Errorf("proposal contains unknown, duplicate or grouped order")
			}
			if !c.Qualified(person, order, w.Work[order.WorkTypeCode]) || !c.Enough(person.EquipmentCapacity, order.RequiredEquipment) {
				return dto.Plan{}, fmt.Errorf("proposal violates staff eligibility")
			}
			seen[id] = true
		}
	}
	return buildSeeded(ctx, w, seed)
}

// Quality is computed from visit IDs and the authoritative world, rather than
// trusting route totals or solver objective values.
func Quality(w *c.World, p dto.Plan) dto.PlanQuality {
	q := dto.PlanQuality{}
	for _, r := range p.Routes {
		for i, s := range r.Stops {
			if s.Type == "VISIT" && s.OrderId != nil {
				q.Visits++
				q.WeightedVisits++
				if w.Orders[*s.OrderId].IsRepeat {
					q.WeightedVisits++
				}
			}
			if i > 0 && s.Type != "BREAK" {
				seconds, _, _ := w.Matrix.Travel(r.Stops[i-1].LocationId, s.LocationId)
				q.TravelSec += seconds
			}
		}
	}
	if p.Summary != nil && p.Summary.UtilizationCv != nil {
		q.UtilizationCV = *p.Summary.UtilizationCv
	}
	return q
}
func Better(a, b dto.PlanQuality) bool {
	if a.WeightedVisits != b.WeightedVisits {
		return a.WeightedVisits > b.WeightedVisits
	}
	if a.Visits != b.Visits {
		return a.Visits > b.Visits
	}
	if a.TravelSec != b.TravelSec {
		return a.TravelSec < b.TravelSec
	}
	return a.UtilizationCV+1e-9 < b.UtilizationCV
}
