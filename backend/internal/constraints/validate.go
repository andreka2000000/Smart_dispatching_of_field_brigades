// Package constraints validates plans independently of the scheduling heuristic.
package constraints

import (
	"fmt"
	"magistral/backend/internal/domain"
	"magistral/backend/internal/dto"
	"time"
)

type Violation struct{ Code, Message string }

func (v *Violation) Error() string { return v.Code + ": " + v.Message }
func bad(code, format string, args ...any) error {
	return &Violation{code, fmt.Sprintf(format, args...)}
}
func Time(t dto.DateTime) time.Time  { x, _ := time.Parse(time.RFC3339, string(t)); return x }
func Stamp(t time.Time) dto.DateTime { return dto.DateTime(t.In(domain.Location).Format(time.RFC3339)) }
func Local(date string, t dto.LocalTime) time.Time {
	x, _ := time.ParseInLocation("2006-01-02 15:04", date+" "+string(t), domain.Location)
	return x
}
func Has[T comparable](values []T, v T) bool {
	for _, x := range values {
		if x == v {
			return true
		}
	}
	return false
}
func Qualified(s dto.Staff, o dto.Order, w dto.WorkType) bool {
	profile := false
	for _, p := range s.Profiles {
		if Has(w.Profiles, p) {
			profile = true
		}
	}
	return profile && Has(s.Technologies, o.Technology) && (!o.RequiresHeight || s.HeightPermit)
}
func Enough(e, need dto.Equipment) bool {
	return e.ONT >= need.ONT && e.ROUTER >= need.ROUTER && e.STB >= need.STB
}
func Consume(e, n dto.Equipment) dto.Equipment {
	return dto.Equipment{ONT: e.ONT - n.ONT, ROUTER: e.ROUTER - n.ROUTER, STB: e.STB - n.STB}
}

type World struct {
	Day       dto.DayInput
	Reference dto.ReferenceData
	Matrix    *domain.Matrix
	Staff     map[string]dto.Staff
	Orders    map[string]dto.Order
	Work      map[dto.WorkTypeCode]dto.WorkType
	Centers   map[string]dto.ServiceCenter
	// Event processing will supply these facts; the validator never infers them
	// from a proposed plan or silently approves changed history.
	Previous     *dto.Plan
	EndOverrides map[string]dto.DateTime
	NewOrderAt   map[string]dto.DateTime
}

func New(r dto.ReferenceData, d dto.DayInput, m dto.TravelMatrix) (*World, error) {
	matrix, e := domain.NewMatrix(m)
	if e != nil {
		return nil, e
	}
	w := &World{Day: d, Reference: r, Matrix: matrix, Staff: map[string]dto.Staff{}, Orders: map[string]dto.Order{}, Work: map[dto.WorkTypeCode]dto.WorkType{}, Centers: map[string]dto.ServiceCenter{}}
	for _, s := range r.Staff {
		w.Staff[s.StaffId] = s
	}
	for _, o := range d.Orders {
		w.Orders[o.OrderId] = o
	}
	for _, x := range r.WorkTypes {
		w.Work[x.Code] = x
	}
	for _, c := range r.ServiceCenters {
		w.Centers[c.ServiceCenterId] = c
	}
	return w, nil
}
func (w *World) Route(r dto.Route) error {
	s, ok := w.Staff[r.StaffId]
	if !ok || Has(w.Day.AbsentStaffIds, r.StaffId) {
		return bad("H2", "unknown or absent staff %s", r.StaffId)
	}
	if r.ServiceCenterId != s.ServiceCenterId {
		return bad("H8", "wrong home center")
	}
	if r.Status != "ACTIVE" && r.Status != "UNAVAILABLE" {
		return bad("H2", "invalid route status")
	}
	if r.Status == "UNAVAILABLE" && r.UnavailableFrom == nil {
		return bad("H2", "missing unavailable_from")
	}
	if len(r.Stops) == 0 {
		return nil
	}
	if r.Stops[0].Type != "DEPOT_START" || (r.Status != "UNAVAILABLE" && r.Stops[len(r.Stops)-1].Type != "DEPOT_END") {
		return bad("H8", "missing depot endpoints")
	}
	startShift, endShift := Local(w.Day.Date, s.Shift.Start), Local(w.Day.Date, s.Shift.End)
	equipment := s.EquipmentStart
	breaks, visits := 0, 0
	for i, x := range r.Stops {
		a, b, c := Time(x.Arrival), Time(x.Start), Time(x.End)
		if a.IsZero() || b.IsZero() || c.IsZero() || a.Nanosecond() != 0 || b.Nanosecond() != 0 || c.Nanosecond() != 0 {
			return bad("H6", "invalid timestamp")
		}
		if x.Seq != int64(i+1) || b.Before(a) || c.Before(b) {
			return bad("H6", "invalid stop sequence or chronology")
		}
		if a.Before(startShift) || c.After(endShift) {
			return bad("H2", "stop outside shift")
		}
		if r.UnavailableFrom != nil && !b.Before(Time(*r.UnavailableFrom)) && (r.UnavailableUntil == nil || b.Before(Time(*r.UnavailableUntil))) {
			return bad("H2", "stop during unavailability")
		}
		if i > 0 {
			prev := r.Stops[i-1]
			travel, _, e := w.Matrix.Travel(prev.LocationId, x.LocationId)
			if e != nil {
				return bad("H6", "unknown matrix point")
			}
			if x.Type == "BREAK" {
				travel = 0
				if x.LocationId != prev.LocationId {
					return bad("H3", "break must stay at preceding location")
				}
			}
			if a.Before(Time(prev.End).Add(time.Duration(travel) * time.Second)) {
				return bad("H6", "arrival violates road time")
			}
		}
		if x.Type != "VISIT" && x.OrderId != nil {
			return bad("H9", "non-visit has order_id")
		}
		switch x.Type {
		case "DEPOT_START", "DEPOT_END":
			if x.LocationId != s.ServiceCenterId || !a.Equal(b) || !b.Equal(c) || (x.Type == "DEPOT_START" && i != 0) || (x.Type == "DEPOT_END" && i != len(r.Stops)-1) {
				return bad("H8", "invalid depot stop")
			}
		case "VISIT":
			visits++
			if x.OrderId == nil {
				return bad("H9", "visit missing order")
			}
			o, ok := w.Orders[*x.OrderId]
			if !ok {
				return bad("H9", "unknown order")
			}
			work, ok := w.Work[o.WorkTypeCode]
			if !ok || !Qualified(s, o, work) {
				return bad("H1", "staff not qualified for %s", o.OrderId)
			}
			if x.LocationId != o.BuildingId {
				return bad("H6", "visit at wrong building")
			}
			if b.Before(Time(o.Slot.Start)) || b.After(Time(o.LatestStart)) {
				return bad("H4", "visit outside allowed window")
			}
			expected := b.Add(time.Duration(work.DurationMin) * time.Minute)
			if override, ok := w.EndOverrides[o.OrderId]; ok {
				expected = Time(override)
			}
			if !c.Equal(expected) {
				return bad("H5", "wrong visit duration")
			}
			if !Enough(equipment, o.RequiredEquipment) {
				return bad("H7", "negative equipment")
			}
			equipment = Consume(equipment, o.RequiredEquipment)
			if at, ok := w.NewOrderAt[o.OrderId]; ok && b.Before(Time(at)) {
				return bad("H10", "visit precedes NEW_ORDER")
			}
		case "BREAK":
			breaks++
			if i == 0 || b.Before(Local(w.Day.Date, s.Shift.BreakWindow.From)) || b.After(Local(w.Day.Date, s.Shift.BreakWindow.To)) || c.Sub(b) < time.Duration(s.Shift.BreakDurationMin)*time.Minute {
				return bad("H3", "invalid break")
			}
		case "RESTOCK":
			center, ok := w.Centers[x.LocationId]
			if !ok || b.Before(Local(w.Day.Date, center.OpeningHours.Open)) || c.After(Local(w.Day.Date, center.OpeningHours.Close)) || c.Sub(b) < time.Duration(center.RestockDurationMin)*time.Minute {
				return bad("H8", "restock at closed or unknown center")
			}
			equipment = s.EquipmentCapacity
		default:
			return bad("H6", "unknown stop type")
		}
	}
	if (visits > 0 && breaks != 1) || (visits == 0 && breaks > 1) {
		return bad("H3", "expected one break for working route")
	}
	return nil
}
func (w *World) Validate(p dto.Plan) error {
	if p.Date != w.Day.Date || !Time(p.AsOf).Equal(Time(w.Day.Now)) || p.PlanVersion < 1 {
		return bad("H10", "plan date/time/version mismatch")
	}
	counts := map[string]int{}
	routes := map[string]dto.Route{}
	add := func(id string) error {
		if _, ok := w.Orders[id]; !ok {
			return bad("H9", "unknown order %s", id)
		}
		counts[id]++
		return nil
	}
	for _, r := range p.Routes {
		if _, ok := routes[r.StaffId]; ok {
			return bad("H9", "duplicate route")
		}
		routes[r.StaffId] = r
		if e := w.Route(r); e != nil {
			return e
		}
		for _, x := range r.Stops {
			if x.Type == "VISIT" {
				if e := add(*x.OrderId); e != nil {
					return e
				}
			}
		}
	}
	for id := range w.Staff {
		if !Has(w.Day.AbsentStaffIds, id) {
			if _, ok := routes[id]; !ok {
				return bad("H9", "missing staff route")
			}
		}
	}
	for _, u := range p.Unassigned {
		if e := add(u.OrderId); e != nil {
			return e
		}
	}
	for _, g := range p.PgpGroups {
		for _, id := range g.OrderIds {
			if e := add(id); e != nil {
				return e
			}
		}
	}
	for _, c := range p.Cancelled {
		if e := add(c.OrderId); e != nil {
			return e
		}
	}
	for id := range w.Orders {
		if counts[id] != 1 {
			return bad("H9", "order %s accounted %d times", id, counts[id])
		}
	}
	now := Time(p.AsOf)
	if w.Previous == nil {
		for _, r := range p.Routes {
			for _, x := range r.Stops {
				if Time(x.Start).Before(now) {
					return bad("H10", "initial plan starts before now")
				}
			}
		}
		return nil
	}
	oldRoutes := map[string]dto.Route{}
	for _, r := range w.Previous.Routes {
		oldRoutes[r.StaffId] = r
		nr := routes[r.StaffId]
		for i, x := range r.Stops {
			if !Time(x.Start).Before(now) {
				continue
			}
			if i >= len(nr.Stops) {
				return bad("H10", "started stop removed")
			}
			y := nr.Stops[i]
			sameOrder := (x.OrderId == nil && y.OrderId == nil) || (x.OrderId != nil && y.OrderId != nil && *x.OrderId == *y.OrderId)
			end := x.End
			if x.OrderId != nil {
				if at, ok := w.EndOverrides[*x.OrderId]; ok {
					end = at
				}
			}
			if x.Type != y.Type || !sameOrder || x.LocationId != y.LocationId || !Time(x.Arrival).Equal(Time(y.Arrival)) || !Time(x.Start).Equal(Time(y.Start)) || !Time(end).Equal(Time(y.End)) {
				return bad("H10", "started stop changed")
			}
		}
	}
	for _, r := range p.Routes {
		old := oldRoutes[r.StaffId]
		for i, x := range r.Stops {
			if Time(x.Start).Before(now) {
				if i >= len(old.Stops) || !sameStop(x, old.Stops[i]) || !Time(x.Start).Equal(Time(old.Stops[i].Start)) {
					return bad("H10", "new stop backdated before model time")
				}
				continue
			}
			unchanged := false
			for j, y := range old.Stops {
				if sameStop(x, y) {
					if i == 0 && j == 0 {
						unchanged = true
					}
					if i > 0 && j > 0 && sameStop(r.Stops[i-1], old.Stops[j-1]) {
						unchanged = true
					}
				}
			}
			if unchanged {
				continue
			}
			if i == 0 {
				if Time(x.Start).Before(now) {
					return bad("H10", "new departure before now")
				}
				continue
			}
			prev := r.Stops[i-1]
			depart := Time(prev.End)
			if now.After(depart) {
				depart = now
			}
			travel, _, e := w.Matrix.Travel(prev.LocationId, x.LocationId)
			if e != nil {
				return bad("H10", "unknown road point")
			}
			if x.Type == "BREAK" {
				travel = 0
			}
			if Time(x.Arrival).Before(depart.Add(time.Duration(travel) * time.Second)) {
				return bad("H10", "new arrival precedes notification plus road time")
			}
		}
	}
	return nil
}
func sameStop(a, b dto.Stop) bool {
	if a.Type != b.Type || a.LocationId != b.LocationId {
		return false
	}
	if a.OrderId == nil || b.OrderId == nil {
		return a.OrderId == nil && b.OrderId == nil
	}
	return *a.OrderId == *b.OrderId
}
