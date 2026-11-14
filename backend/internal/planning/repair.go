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

// tail rebuilds only the unfrozen suffix. Every changed leg departs no earlier
// than model now; no vehicle is teleported onto the old road geometry.
func tail(w *c.World, s dto.Staff, ids []string, breakAt int, prefix []dto.Stop) (dto.Route, bool) {
	r := dto.Route{StaffId: s.StaffId, ServiceCenterId: s.ServiceCenterId, Status: "ACTIVE", Stops: append([]dto.Stop{}, prefix...)}
	periods := w.Unavailable[s.StaffId]
	var resume time.Time
	terminal := false
	for _, u := range periods {
		r.UnavailableFrom = ptr(u.From)
		r.UnavailableUntil = u.Until
		if u.Until == nil {
			terminal = true
			r.Status = "UNAVAILABLE"
		} else {
			if c.Time(*u.Until).After(resume) {
				resume = c.Time(*u.Until)
			}
			if c.Time(*u.Until).After(c.Time(w.Day.Now)) {
				r.Status = "UNAVAILABLE"
			}
		}
	}
	if len(prefix) > 0 && prefix[len(prefix)-1].Type == "DEPOT_END" {
		return r, len(ids) == 0 && w.Route(r) == nil
	}
	equipment := s.EquipmentStart
	breaks, visits := 0, 0
	for _, x := range prefix {
		switch x.Type {
		case "RESTOCK":
			equipment = s.EquipmentCapacity
		case "VISIT":
			visits++
			equipment = c.Consume(equipment, w.Orders[*x.OrderId].RequiredEquipment)
		case "BREAK":
			breaks++
		}
	}
	if terminal {
		if len(ids) > 0 {
			return r, false
		}
		return r, w.Route(r) == nil
	}
	if len(prefix) == 0 && len(ids) == 0 {
		return r, true
	}
	now := later(c.Time(w.Day.Now), c.Local(w.Day.Date, s.Shift.Start))
	loc := s.ServiceCenterId
	shiftEnd := c.Local(w.Day.Date, s.Shift.End)
	if len(prefix) > 0 {
		last := prefix[len(prefix)-1]
		now = later(now, c.Time(last.End))
		loc = last.LocationId
	}
	add := func(kind, to string, order *string, a, b, end time.Time, sec, dist int64) {
		r.Stops = append(r.Stops, dto.Stop{Seq: int64(len(r.Stops) + 1), Type: kind, LocationId: to, OrderId: order, Arrival: c.Stamp(a), Start: c.Stamp(b), End: c.Stamp(end), TravelSec: ptr(sec), DistanceM: ptr(dist), EquipmentAfter: ptr(equipment)})
		loc = to
		now = end
	}
	if len(prefix) == 0 {
		now = later(now, resume)
		if now.After(shiftEnd) {
			return r, false
		}
		add("DEPOT_START", loc, nil, now, now, now, 0, 0)
	} else if !resume.IsZero() && c.Time(prefix[len(prefix)-1].End).Before(resume) {
		// After finite unavailability the employee resumes from the home center.
		// An explicit home stop keeps the road and equipment model auditable.
		center := w.Centers[s.ServiceCenterId]
		sec, dist, e := w.Matrix.Travel(loc, s.ServiceCenterId)
		if e != nil {
			return r, false
		}
		arrival := later(now.Add(time.Duration(sec)*time.Second), resume)
		start := later(arrival, c.Local(w.Day.Date, center.OpeningHours.Open))
		end := start.Add(time.Duration(center.RestockDurationMin) * time.Minute)
		if end.After(c.Local(w.Day.Date, center.OpeningHours.Close)) || end.After(shiftEnd) {
			return r, false
		}
		equipment = s.EquipmentCapacity
		add("RESTOCK", s.ServiceCenterId, nil, arrival, start, end, sec, dist)
	}
	needBreak := breaks == 0 && (visits > 0 || len(ids) > 0)
	for i := 0; i <= len(ids); i++ {
		if needBreak && i == breakAt {
			start := later(now, c.Local(w.Day.Date, s.Shift.BreakWindow.From))
			end := start.Add(time.Duration(s.Shift.BreakDurationMin) * time.Minute)
			if start.After(c.Local(w.Day.Date, s.Shift.BreakWindow.To)) || end.After(shiftEnd) {
				return r, false
			}
			add("BREAK", loc, nil, now, start, end, 0, 0)
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
			found := false
			var best dto.ServiceCenter
			var ba, bs, be, bv time.Time
			var bt, bd int64
			for _, center := range w.Reference.ServiceCenters {
				sec, dist, e := w.Matrix.Travel(loc, center.ServiceCenterId)
				if e != nil {
					continue
				}
				a := now.Add(time.Duration(sec) * time.Second)
				b := later(a, c.Local(w.Day.Date, center.OpeningHours.Open))
				end := b.Add(time.Duration(center.RestockDurationMin) * time.Minute)
				if end.After(c.Local(w.Day.Date, center.OpeningHours.Close)) || end.After(shiftEnd) {
					continue
				}
				leg, _, e := w.Matrix.Travel(center.ServiceCenterId, o.BuildingId)
				if e != nil {
					continue
				}
				v := end.Add(time.Duration(leg) * time.Second)
				if !found || v.Before(bv) || (v.Equal(bv) && center.ServiceCenterId < best.ServiceCenterId) {
					found = true
					best = center
					ba = a
					bs = b
					be = end
					bv = v
					bt = sec
					bd = dist
				}
			}
			if !found {
				return r, false
			}
			equipment = s.EquipmentCapacity
			add("RESTOCK", best.ServiceCenterId, nil, ba, bs, be, bt, bd)
		}
		sec, dist, e := w.Matrix.Travel(loc, o.BuildingId)
		if e != nil {
			return r, false
		}
		a := now.Add(time.Duration(sec) * time.Second)
		b := later(a, c.Time(o.Slot.Start))
		end := b.Add(time.Duration(work.DurationMin) * time.Minute)
		if b.After(c.Time(o.LatestStart)) || end.After(c.Time(o.Deadline)) || end.After(shiftEnd) {
			return r, false
		}
		equipment = c.Consume(equipment, o.RequiredEquipment)
		add("VISIT", o.BuildingId, ptr(o.OrderId), a, b, end, sec, dist)
	}
	sec, dist, e := w.Matrix.Travel(loc, s.ServiceCenterId)
	if e != nil {
		return r, false
	}
	end := now.Add(time.Duration(sec) * time.Second)
	if end.After(shiftEnd) {
		return r, false
	}
	add("DEPOT_END", s.ServiceCenterId, nil, end, end, end, sec, dist)
	return r, w.Route(r) == nil
}
func bestTail(ctx context.Context, w *c.World, s dto.Staff, ids []string, prefix []dto.Stop) (dto.Route, bool) {
	var best dto.Route
	found := false
	for b := 0; b <= len(ids); b++ {
		if ctx.Err() != nil {
			break
		}
		r, ok := tail(w, s, ids, b, prefix)
		if ok && (!found || c.Time(r.Stops[len(r.Stops)-1].End).Before(c.Time(best.Stops[len(best.Stops)-1].End))) {
			best = r
			found = true
		}
	}
	return best, found
}

// Repair retains feasible routes verbatim and repairs affected suffixes. pinned
// includes a delayed/transferred visit that starts exactly at the event time.
func Repair(ctx context.Context, w *c.World, old dto.Plan, cancelled map[string]dto.CancelledOrder, pinned map[string]bool) (dto.Plan, error) {
	rawPlan := old // copy all slices before mutating; old is the validator's history.
	p := rawPlan
	p.Routes = append([]dto.Route{}, old.Routes...)
	p.Unassigned = []dto.UnassignedOrder{}
	p.Cancelled = []dto.CancelledOrder{}
	p.Alerts = append([]dto.Alert{}, old.Alerts...)
	p.PgpGroups = append([]dto.PgpGroup{}, old.PgpGroups...)
	p.AsOf = w.Day.Now
	p.PlanVersion = old.PlanVersion + 1
	prefixes := make([][]dto.Stop, len(p.Routes))
	sequences := make([][]string, len(p.Routes))
	assigned := map[string]bool{}
	excluded := map[string]bool{}
	for id := range cancelled {
		excluded[id] = true
	}
	for _, g := range p.PgpGroups {
		for _, id := range g.OrderIds {
			excluded[id] = true
		}
	}
	for i, r := range old.Routes {
		limit := 0
		for j, x := range r.Stops {
			if c.Time(x.Start).Before(c.Time(w.Day.Now)) || (x.OrderId != nil && pinned[*x.OrderId]) {
				limit = j + 1
			}
		}
		prefix := append([]dto.Stop{}, r.Stops[:limit]...)
		for j := range prefix {
			if id := prefix[j].OrderId; id != nil {
				if end, ok := w.EndOverrides[*id]; ok {
					prefix[j].End = end
				}
			}
		}
		prefixes[i] = prefix
		ids := []string{}
		candidate := r
		candidate.Stops = append([]dto.Stop{}, r.Stops...)
		mustRepair := false
		for j, x := range candidate.Stops {
			if x.OrderId != nil {
				if end, ok := w.EndOverrides[*x.OrderId]; ok {
					candidate.Stops[j].End = end
				}
				if j >= limit && excluded[*x.OrderId] {
					mustRepair = true
				}
				if j >= limit && !excluded[*x.OrderId] {
					ids = append(ids, *x.OrderId)
				}
			}
		}
		// Metadata reflects the most recently recorded period; the validator also
		// checks all persisted periods, so an older outage cannot be forgotten.
		candidate.Status = "ACTIVE"
		for _, u := range w.Unavailable[r.StaffId] {
			candidate.UnavailableFrom = ptr(u.From)
			candidate.UnavailableUntil = u.Until
			if u.Until == nil || c.Time(*u.Until).After(c.Time(w.Day.Now)) {
				candidate.Status = "UNAVAILABLE"
				mustRepair = true
			}
		}
		var chosen dto.Route
		if !mustRepair && w.Route(candidate) == nil {
			chosen = candidate
		} else {
			// First recover a valid suffix, removing future visits if necessary. This
			// mandatory recovery is bounded by the old route length, not optional search.
			found := false
			for n := len(ids); n >= 0 && !found; n-- {
				for b := 0; b <= n; b++ {
					if ctx.Err() != nil {
						return dto.Plan{}, ctx.Err()
					}
					if v, ok := tail(w, w.Staff[r.StaffId], ids[:n], b, prefix); ok {
						chosen = v
						ids = ids[:n]
						found = true
						break
					}
				}
			}
			if !found {
				return dto.Plan{}, fmt.Errorf("fixed history of %s cannot satisfy constraints after event", r.StaffId)
			}
		}
		p.Routes[i] = chosen
		sequences[i] = ids
		for _, x := range chosen.Stops {
			if x.OrderId != nil {
				assigned[*x.OrderId] = true
			}
		}
	}
	pending := []dto.Order{}
	for _, o := range w.Day.Orders {
		if !assigned[o.OrderId] && !excluded[o.OrderId] {
			pending = append(pending, o)
		}
	}
	sort.Slice(pending, func(i, j int) bool {
		a, b := pending[i], pending[j]
		if a.IsRepeat != b.IsRepeat {
			return a.IsRepeat
		}
		if a.LatestStart != b.LatestStart {
			return c.Time(a.LatestStart).Before(c.Time(b.LatestStart))
		}
		return a.OrderId < b.OrderId
	})
	for _, o := range pending {
		if ctx.Err() != nil {
			break
		}
		bestIndex := -1
		var best dto.Route
		var bestIDs []string
		bestEnd := time.Time{}
		for i, r := range p.Routes {
			if !c.Qualified(w.Staff[r.StaffId], o, w.Work[o.WorkTypeCode]) {
				continue
			}
			for pos := 0; pos <= len(sequences[i]); pos++ {
				if ctx.Err() != nil {
					break
				}
				ids := append([]string{}, sequences[i][:pos]...)
				ids = append(ids, o.OrderId)
				ids = append(ids, sequences[i][pos:]...)
				v, ok := bestTail(ctx, w, w.Staff[r.StaffId], ids, prefixes[i])
				if !ok {
					continue
				}
				end := c.Time(v.Stops[len(v.Stops)-1].End)
				if bestIndex < 0 || end.Before(bestEnd) {
					bestIndex = i
					best = v
					bestIDs = ids
					bestEnd = end
				}
			}
		}
		if bestIndex >= 0 {
			p.Routes[bestIndex] = best
			sequences[bestIndex] = bestIDs
			assigned[o.OrderId] = true
		}
	}
	since := map[string]*dto.DateTime{}
	for _, u := range old.Unassigned {
		since[u.OrderId] = u.Since
	}
	for _, o := range w.Day.Orders {
		if assigned[o.OrderId] || excluded[o.OrderId] {
			continue
		}
		code, text := reason(w, o)
		at := since[o.OrderId]
		if at == nil {
			at = ptr(w.Day.Now)
		}
		p.Unassigned = append(p.Unassigned, dto.UnassignedOrder{OrderId: o.OrderId, ReasonCode: code, ReasonText: text, SuggestedAction: "ESCALATE", Since: at})
	}
	for _, x := range cancelled {
		p.Cancelled = append(p.Cancelled, x)
	}
	sort.Slice(p.Cancelled, func(i, j int) bool { return p.Cancelled[i].OrderId < p.Cancelled[j].OrderId })
	Refresh(w, &p)
	if e := w.Validate(p); e != nil {
		return dto.Plan{}, e
	}
	return p, nil
}

// Refresh recalculates display fields from authoritative data, including frozen
// visits' overridden ends. These fields are not used as validation evidence.
func Refresh(w *c.World, p *dto.Plan) {
	var visits, travel, pgp int64
	var mean, variance float64
	for i := range p.Routes {
		r := &p.Routes[i]
		stock := w.Staff[r.StaffId].EquipmentStart
		var v, t, d, work int64
		for j := range r.Stops {
			x := &r.Stops[j]
			if x.Type == "VISIT" {
				v++
				stock = c.Consume(stock, w.Orders[*x.OrderId].RequiredEquipment)
				work += int64(c.Time(x.End).Sub(c.Time(x.Start)).Seconds())
			}
			if x.Type == "RESTOCK" {
				stock = w.Staff[r.StaffId].EquipmentCapacity
			}
			x.EquipmentAfter = ptr(stock)
			sec, dist := int64(0), int64(0)
			if j > 0 && x.Type != "BREAK" {
				sec, dist, _ = w.Matrix.Travel(r.Stops[j-1].LocationId, x.LocationId)
			}
			x.TravelSec = ptr(sec)
			x.DistanceM = ptr(dist)
			t += sec
			d += dist
			status := "PLANNED"
			if !c.Time(x.End).After(c.Time(p.AsOf)) {
				status = "DONE"
			} else if !c.Time(x.Start).After(c.Time(p.AsOf)) {
				status = "IN_PROGRESS"
			}
			x.Status = ptr(status)
		}
		s := w.Staff[r.StaffId]
		available := c.Local(w.Day.Date, s.Shift.End).Sub(c.Local(w.Day.Date, s.Shift.Start)).Seconds() - float64(s.Shift.BreakDurationMin*60)
		u := 0.0
		if available > 0 {
			u = float64(work+t) / available
		}
		r.Totals = &dto.RouteTotals{Visits: ptr(v), TravelSec: ptr(t), DistanceM: ptr(d), WorkMin: ptr(work / 60), Utilization: ptr(u)}
		visits += v
		travel += t
		mean += u
	}
	n := float64(len(p.Routes))
	if n > 0 {
		mean /= n
		for _, r := range p.Routes {
			delta := *r.Totals.Utilization - mean
			variance += delta * delta
		}
		variance /= n
	}
	cv := 0.0
	if mean > 0 {
		cv = math.Sqrt(variance) / mean
	}
	for _, g := range p.PgpGroups {
		pgp += int64(len(g.OrderIds))
	}
	p.Summary = &dto.PlanSummary{OrdersTotal: ptr(int64(len(w.Day.Orders))), VisitsPlanned: ptr(visits), Unassigned: ptr(int64(len(p.Unassigned))), PgpOrders: ptr(pgp), Cancelled: ptr(int64(len(p.Cancelled))), TravelSecTotal: ptr(travel), UtilizationCv: ptr(cv)}
}
