// Package planning implements a deterministic insertion heuristic, not an
// optimal VRPTW solver. Every returned plan is checked by constraints.
package planning

import (
	"context"
	"fmt"
	c "magistral/backend/internal/constraints"
	"magistral/backend/internal/dto"
	"math"
	"sort"
	"time"
)

func ptr[T any](v T) *T { return &v }
func later(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

// schedule simulates the entire route, including its return leg. breakAt is a
// visit boundary; restocks are inserted when the next visit needs more stock.
func schedule(w *c.World, s dto.Staff, ids []string, breakAt int) (dto.Route, bool) {
	r := dto.Route{StaffId: s.StaffId, ServiceCenterId: s.ServiceCenterId, Status: "ACTIVE", Stops: []dto.Stop{}}
	if len(ids) == 0 {
		return r, true
	}
	now := later(c.Time(w.Day.Now), c.Local(w.Day.Date, s.Shift.Start))
	endShift := c.Local(w.Day.Date, s.Shift.End)
	if now.After(endShift) {
		return r, false
	}
	loc := s.ServiceCenterId
	equipment := s.EquipmentStart
	var travelTotal, distanceTotal, workTotal int64
	appendStop := func(kind, to string, order *string, arrival, start, end time.Time, travel, distance int64) {
		r.Stops = append(r.Stops, dto.Stop{Seq: int64(len(r.Stops) + 1), Type: kind, LocationId: to, OrderId: order, Arrival: c.Stamp(arrival), Start: c.Stamp(start), End: c.Stamp(end), TravelSec: ptr(travel), DistanceM: ptr(distance), EquipmentAfter: ptr(equipment), Status: ptr("PLANNED")})
		now = end
		loc = to
		travelTotal += travel
		distanceTotal += distance
	}
	appendStop("DEPOT_START", loc, nil, now, now, now, 0, 0)
	for i := 0; i <= len(ids); i++ {
		if i == breakAt {
			start := later(now, c.Local(w.Day.Date, s.Shift.BreakWindow.From))
			end := start.Add(time.Duration(s.Shift.BreakDurationMin) * time.Minute)
			if start.After(c.Local(w.Day.Date, s.Shift.BreakWindow.To)) || end.After(endShift) {
				return r, false
			}
			appendStop("BREAK", loc, nil, now, start, end, 0, 0)
		}
		if i == len(ids) {
			break
		}
		o := w.Orders[ids[i]]
		work := w.Work[o.WorkTypeCode]
		if !c.Qualified(s, o, work) {
			return r, false
		}
		if !c.Enough(equipment, o.RequiredEquipment) {
			if !c.Enough(s.EquipmentCapacity, o.RequiredEquipment) {
				return r, false
			}
			// Pick the center yielding the earliest arrival at this visit. Enumerating
			// all restock combinations is intentionally outside this baseline.
			var best *dto.ServiceCenter
			var bestArrival, bestStart, bestEnd, bestVisit time.Time
			var bt, bd int64
			for j := range w.Reference.ServiceCenters {
				center := &w.Reference.ServiceCenters[j]
				sec, dist, e := w.Matrix.Travel(loc, center.ServiceCenterId)
				if e != nil {
					continue
				}
				arrival := now.Add(time.Duration(sec) * time.Second)
				start := later(arrival, c.Local(w.Day.Date, center.OpeningHours.Open))
				end := start.Add(time.Duration(center.RestockDurationMin) * time.Minute)
				if end.After(c.Local(w.Day.Date, center.OpeningHours.Close)) || end.After(endShift) {
					continue
				}
				leg, _, e := w.Matrix.Travel(center.ServiceCenterId, o.BuildingId)
				if e != nil {
					continue
				}
				visit := end.Add(time.Duration(leg) * time.Second)
				if best == nil || visit.Before(bestVisit) || (visit.Equal(bestVisit) && center.ServiceCenterId < best.ServiceCenterId) {
					best = center
					bestArrival = arrival
					bestStart = start
					bestEnd = end
					bestVisit = visit
					bt = sec
					bd = dist
				}
			}
			if best == nil {
				return r, false
			}
			equipment = s.EquipmentCapacity
			appendStop("RESTOCK", best.ServiceCenterId, nil, bestArrival, bestStart, bestEnd, bt, bd)
		}
		sec, dist, e := w.Matrix.Travel(loc, o.BuildingId)
		if e != nil {
			return r, false
		}
		arrival := now.Add(time.Duration(sec) * time.Second)
		start := later(arrival, c.Time(o.Slot.Start))
		end := start.Add(time.Duration(work.DurationMin) * time.Minute)
		if start.After(c.Time(o.LatestStart)) || end.After(c.Time(o.Deadline)) || end.After(endShift) {
			return r, false
		}
		equipment = c.Consume(equipment, o.RequiredEquipment)
		appendStop("VISIT", o.BuildingId, ptr(o.OrderId), arrival, start, end, sec, dist)
		workTotal += work.DurationMin
	}
	sec, dist, e := w.Matrix.Travel(loc, s.ServiceCenterId)
	if e != nil {
		return r, false
	}
	end := now.Add(time.Duration(sec) * time.Second)
	if end.After(endShift) {
		return r, false
	}
	appendStop("DEPOT_END", s.ServiceCenterId, nil, end, end, end, sec, dist)
	available := endShift.Sub(c.Local(w.Day.Date, s.Shift.Start)).Seconds() - float64(s.Shift.BreakDurationMin*60)
	util := 0.0
	if available > 0 {
		util = float64(workTotal*60+travelTotal) / available
	}
	r.Totals = &dto.RouteTotals{Visits: ptr(int64(len(ids))), TravelSec: ptr(travelTotal), DistanceM: ptr(distanceTotal), WorkMin: ptr(workTotal), Utilization: ptr(util)}
	if w.Route(r) != nil {
		return r, false
	}
	return r, true
}
func bestRoute(ctx context.Context, w *c.World, s dto.Staff, ids []string) (dto.Route, bool) {
	var best dto.Route
	found := false
	for b := 0; b <= len(ids); b++ {
		if ctx.Err() != nil {
			break
		}
		r, ok := schedule(w, s, ids, b)
		if ok && (!found || *r.Totals.TravelSec < *best.Totals.TravelSec || (*r.Totals.TravelSec == *best.Totals.TravelSec && c.Time(r.Stops[len(r.Stops)-1].End).Before(c.Time(best.Stops[len(best.Stops)-1].End)))) {
			best = r
			found = true
		}
	}
	return best, found
}
func reason(w *c.World, o dto.Order) (string, string) {
	qualified, capacity := false, false
	for _, s := range w.Reference.Staff {
		if c.Has(w.Day.AbsentStaffIds, s.StaffId) {
			continue
		}
		if !c.Qualified(s, o, w.Work[o.WorkTypeCode]) {
			continue
		}
		qualified = true
		if c.Enough(s.EquipmentCapacity, o.RequiredEquipment) {
			capacity = true
		}
	}
	if !qualified {
		return "NO_QUALIFIED_STAFF", "На смене нет сотрудника с необходимым профилем, технологией и допуском."
	}
	if !capacity {
		return "NO_EQUIPMENT", "Требуемое оборудование превышает вместимость всех подходящих сотрудников."
	}
	// Failure of a heuristic is not a proof of global infeasibility.
	return "OTHER", "Эвристика не нашла допустимую вставку в пределах бюджета; требуется проверка диспетчером."
}
func Build(ctx context.Context, w *c.World) (dto.Plan, error) {
	p := dto.Plan{Date: w.Day.Date, PlanVersion: 1, AsOf: w.Day.Now, Routes: []dto.Route{}, Unassigned: []dto.UnassignedOrder{}, PgpGroups: []dto.PgpGroup{}, Cancelled: []dto.CancelledOrder{}, Alerts: []dto.Alert{}}
	staff := append([]dto.Staff(nil), w.Reference.Staff...)
	sort.Slice(staff, func(i, j int) bool { return staff[i].StaffId < staff[j].StaffId })
	active := []dto.Staff{}
	sequences := [][]string{}
	for _, s := range staff {
		if c.Has(w.Day.AbsentStaffIds, s.StaffId) {
			continue
		}
		active = append(active, s)
		sequences = append(sequences, []string{})
		r, _ := schedule(w, s, nil, 0)
		p.Routes = append(p.Routes, r)
	}
	orders := append([]dto.Order(nil), w.Day.Orders...)
	sort.SliceStable(orders, func(i, j int) bool {
		a, b := orders[i], orders[j]
		if a.IsRepeat != b.IsRepeat {
			return a.IsRepeat
		}
		if !c.Time(a.LatestStart).Equal(c.Time(b.LatestStart)) {
			return c.Time(a.LatestStart).Before(c.Time(b.LatestStart))
		}
		aw := c.Time(a.LatestStart).Sub(c.Time(a.Slot.Start))
		bw := c.Time(b.LatestStart).Sub(c.Time(b.Slot.Start))
		if aw != bw {
			return aw < bw
		}
		return a.OrderId < b.OrderId
	})
	assigned := map[string]bool{}
	for _, o := range orders {
		if ctx.Err() != nil {
			break
		}
		bestIndex := -1
		var best dto.Route
		var bestIDs []string
		score := math.Inf(1)
	search:
		for i, s := range active {
			if !c.Qualified(s, o, w.Work[o.WorkTypeCode]) || !c.Enough(s.EquipmentCapacity, o.RequiredEquipment) {
				continue
			}
			for at := 0; at <= len(sequences[i]); at++ {
				if ctx.Err() != nil {
					break search
				}
				ids := make([]string, 0, len(sequences[i])+1)
				ids = append(ids, sequences[i][:at]...)
				ids = append(ids, o.OrderId)
				ids = append(ids, sequences[i][at:]...)
				r, ok := bestRoute(ctx, w, s, ids)
				if !ok {
					continue
				}
				oldTravel := int64(0)
				if p.Routes[i].Totals != nil {
					oldTravel = *p.Routes[i].Totals.TravelSec
				}
				candidate := float64(*r.Totals.TravelSec-oldTravel) + *r.Totals.Utilization*60
				if candidate < score {
					score = candidate
					bestIndex = i
					best = r
					bestIDs = ids
				}
			}
		}
		if bestIndex >= 0 {
			sequences[bestIndex] = bestIDs
			p.Routes[bestIndex] = best
			assigned[o.OrderId] = true
		}
	}
	for _, o := range orders {
		if !assigned[o.OrderId] {
			code, text := reason(w, o)
			p.Unassigned = append(p.Unassigned, dto.UnassignedOrder{OrderId: o.OrderId, ReasonCode: code, ReasonText: text, SuggestedAction: "ESCALATE", Since: ptr(w.Day.Now)})
		}
	}
	var totalTravel int64
	var mean, variance float64
	for _, r := range p.Routes {
		if r.Totals != nil {
			totalTravel += *r.Totals.TravelSec
			mean += *r.Totals.Utilization
		}
	}
	if len(p.Routes) > 0 {
		mean /= float64(len(p.Routes))
		for _, r := range p.Routes {
			u := 0.0
			if r.Totals != nil {
				u = *r.Totals.Utilization
			}
			variance += (u - mean) * (u - mean)
		}
		variance /= float64(len(p.Routes))
	}
	cv := 0.0
	if mean > 0 {
		cv = math.Sqrt(variance) / mean
	}
	p.Summary = &dto.PlanSummary{OrdersTotal: ptr(int64(len(orders))), VisitsPlanned: ptr(int64(len(assigned))), Unassigned: ptr(int64(len(p.Unassigned))), PgpOrders: ptr(int64(0)), Cancelled: ptr(int64(0)), TravelSecTotal: ptr(totalTravel), UtilizationCv: ptr(cv)}
	if e := w.Validate(p); e != nil {
		return dto.Plan{}, fmt.Errorf("baseline rejected: %w", e)
	}
	return p, nil
}
