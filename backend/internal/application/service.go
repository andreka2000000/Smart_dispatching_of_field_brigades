package application

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx/v5"
	"magistral/backend/internal/domain"
	"magistral/backend/internal/dto"
	"magistral/backend/internal/storage/postgres"
	"reflect"
	"time"
)

type Error struct {
	Status        int
	Code, Message string
}

func (e *Error) Error() string { return e.Message }
func invalid(format string, args ...any) error {
	return &Error{422, "INVALID_DATA", fmt.Sprintf(format, args...)}
}

type Service struct{ Store *postgres.Store }

func unique(seen map[string]bool, id string) error {
	if id == "" || seen[id] {
		return invalid("Empty or duplicate identifier: %s", id)
	}
	seen[id] = true
	return nil
}
func ValidateReference(r dto.ReferenceData) error {
	centers, staff, nodes, works := map[string]bool{}, map[string]bool{}, map[string]bool{}, map[string]bool{}
	buildings := map[string]dto.Building{}
	for _, x := range r.ServiceCenters {
		if e := unique(centers, x.ServiceCenterId); e != nil {
			return e
		}
		if x.OpeningHours.Open >= x.OpeningHours.Close {
			return invalid("Invalid opening hours: %s", x.ServiceCenterId)
		}
	}
	for _, x := range r.Staff {
		if e := unique(staff, x.StaffId); e != nil {
			return e
		}
		if !centers[x.ServiceCenterId] {
			return invalid("Unknown service center for %s", x.StaffId)
		}
		if x.EquipmentStart.ONT > x.EquipmentCapacity.ONT || x.EquipmentStart.ROUTER > x.EquipmentCapacity.ROUTER || x.EquipmentStart.STB > x.EquipmentCapacity.STB {
			return invalid("Equipment exceeds capacity: %s", x.StaffId)
		}
		sh := x.Shift
		if sh.Start >= sh.End || sh.BreakWindow.From > sh.BreakWindow.To || sh.BreakWindow.From < sh.Start || sh.BreakWindow.To > sh.End {
			return invalid("Invalid shift: %s", x.StaffId)
		}
		start, _ := time.Parse("15:04", string(sh.BreakWindow.From))
		end, _ := time.Parse("15:04", string(sh.End))
		if sh.BreakDurationMin > int64(end.Sub(start)/time.Minute) {
			return invalid("Break cannot fit shift: %s", x.StaffId)
		}
	}
	for _, x := range r.AccessNodes {
		if e := unique(nodes, x.AccessNodeId); e != nil {
			return e
		}
		expected := map[string]dto.Technology{"OLT": "PON", "SWITCH": "FTTB", "DSLAM": "XDSL", "CMTS": "DOCSIS"}
		if expected[x.NodeType] != x.Technology {
			return invalid("Node technology mismatch: %s", x.AccessNodeId)
		}
		local := map[string]bool{}
		for _, b := range x.Buildings {
			if e := unique(local, b.BuildingId); e != nil {
				return e
			}
			if old, ok := buildings[b.BuildingId]; ok && !reflect.DeepEqual(old, b) {
				return invalid("Inconsistent repeated building: %s", b.BuildingId)
			}
			if b.Lat < -90 || b.Lat > 90 || b.Lon < -180 || b.Lon > 180 {
				return invalid("Invalid building coordinates")
			}
			if b.ZoneServiceCenterId != nil && !centers[*b.ZoneServiceCenterId] {
				return invalid("Unknown building service center")
			}
			buildings[b.BuildingId] = b
		}
	}
	for _, x := range r.WorkTypes {
		if e := unique(works, string(x.Code)); e != nil {
			return e
		}
		if x.DeadlineKind != "SLOT" && x.DeadlineValue == nil {
			return invalid("Missing deadline_value: %s", x.Code)
		}
	}
	return nil
}
func Coverage(r dto.ReferenceData, m dto.TravelMatrix) error {
	points := map[string]bool{}
	for _, p := range m.Points {
		points[p] = true
	}
	for _, c := range r.ServiceCenters {
		if !points[c.ServiceCenterId] {
			return invalid("Matrix missing %s", c.ServiceCenterId)
		}
	}
	for _, n := range r.AccessNodes {
		for _, b := range n.Buildings {
			if !points[b.BuildingId] {
				return invalid("Matrix missing %s", b.BuildingId)
			}
		}
	}
	return nil
}
func (s *Service) Reference(ctx context.Context, r dto.ReferenceData) error {
	if e := ValidateReference(r); e != nil {
		return e
	}
	return s.Store.Update(ctx, func(tx pgx.Tx) error {
		m, e := postgres.Read[dto.TravelMatrix](ctx, tx, "travel_matrices")
		if e != nil {
			return e
		}
		if m != nil {
			if e = Coverage(r, *m); e != nil {
				return e
			}
		}
		return postgres.ReplaceReference(ctx, tx, r)
	})
}
func (s *Service) Matrix(ctx context.Context, m dto.TravelMatrix) error {
	if _, e := domain.NewMatrix(m); e != nil {
		return invalid("%s", e)
	}
	return s.Store.Update(ctx, func(tx pgx.Tx) error {
		r, e := postgres.Read[dto.ReferenceData](ctx, tx, "reference_data")
		if e != nil {
			return e
		}
		if r != nil {
			if e = Coverage(*r, m); e != nil {
				return e
			}
		}
		if e = postgres.Put(ctx, tx, "travel_matrices", m); e != nil {
			return e
		}
		_, e = tx.Exec(ctx, "DELETE FROM plan_snapshots; UPDATE days SET plan_revision=plan_revision+1,current_version=0")
		return e
	})
}
func ValidateDay(date string, d *dto.DayInput, r dto.ReferenceData) error {
	if date != d.Date {
		return invalid("URL date differs from body date")
	}
	clock, e := domain.NewClock(string(d.Now))
	if e != nil {
		return invalid("Invalid now: %s", e)
	}
	if clock.Now.Format("2006-01-02") != date {
		return invalid("now must belong to day in UTC+05:00")
	}
	d.Now = dto.DateTime(clock.Now.Format(time.RFC3339))
	staff := map[string]bool{}
	for _, x := range r.Staff {
		staff[x.StaffId] = true
	}
	absent := map[string]bool{}
	for _, id := range d.AbsentStaffIds {
		if e := unique(absent, id); e != nil {
			return e
		}
		if !staff[id] {
			return invalid("Unknown absent staff: %s", id)
		}
	}
	works := map[dto.WorkTypeCode]dto.WorkType{}
	for _, x := range r.WorkTypes {
		works[x.Code] = x
	}
	nodes := map[string]dto.AccessNode{}
	for _, n := range r.AccessNodes {
		nodes[n.AccessNodeId] = n
	}
	seen := map[string]bool{}
	for _, o := range d.Orders {
		if e := unique(seen, o.OrderId); e != nil {
			return e
		}
		w, ok := works[o.WorkTypeCode]
		if !ok {
			return invalid("Unknown work type: %s", o.WorkTypeCode)
		}
		n, ok := nodes[o.AccessNodeId]
		if !ok {
			return invalid("Unknown access node: %s", o.AccessNodeId)
		}
		found := false
		for _, b := range n.Buildings {
			if b.BuildingId == o.BuildingId {
				found = true
				if b.BuildingType != o.BuildingType {
					return invalid("Building type mismatch")
				}
			}
		}
		if !found || n.Technology != o.Technology {
			return invalid("Order building/node/technology mismatch: %s", o.OrderId)
		}
		values := []dto.DateTime{o.RegisteredAt, o.Slot.Start, o.Slot.End, o.LatestStart, o.Deadline}
		times := make([]time.Time, len(values))
		for i, v := range values {
			c, e := domain.NewClock(string(v))
			if e != nil {
				return invalid("Invalid order time: %s", o.OrderId)
			}
			times[i] = c.Now
		}
		reg, start, end, latest, deadline := times[0], times[1], times[2], times[3], times[4]
		if reg.After(clock.Now) || start.Format("2006-01-02") != date || !end.After(start) || latest.Before(start) || latest.After(end) || latest.Add(time.Duration(w.DurationMin)*time.Minute).After(deadline) {
			return invalid("Invalid order time window: %s", o.OrderId)
		}
		expected := end
		if w.MaxStartDelayMin != nil {
			candidate := start.Add(time.Duration(*w.MaxStartDelayMin) * time.Minute)
			if candidate.Before(expected) {
				expected = candidate
			}
		}
		if !latest.Equal(expected) {
			return invalid("latest_start inconsistent with work type: %s", o.OrderId)
		}
	}
	if d.AbsentStaffIds == nil {
		d.AbsentStaffIds = []string{}
	}
	return nil
}
func (s *Service) Day(ctx context.Context, date string, d dto.DayInput) error {
	return s.Store.Update(ctx, func(tx pgx.Tx) error {
		r, e := postgres.Read[dto.ReferenceData](ctx, tx, "reference_data")
		if e != nil {
			return e
		}
		if r == nil {
			return &Error{409, "REFERENCE_NOT_LOADED", "Load reference data first"}
		}
		m, e := postgres.Read[dto.TravelMatrix](ctx, tx, "travel_matrices")
		if e != nil {
			return e
		}
		if m == nil {
			return &Error{409, "MATRIX_NOT_LOADED", "Load travel matrix first"}
		}
		if e = ValidateDay(date, &d, *r); e != nil {
			return e
		}
		return postgres.ReplaceDay(ctx, tx, d)
	})
}
func Counts(r dto.ReferenceData) map[string]int {
	b := map[string]bool{}
	for _, n := range r.AccessNodes {
		for _, x := range n.Buildings {
			b[x.BuildingId] = true
		}
	}
	return map[string]int{"service_centers": len(r.ServiceCenters), "staff": len(r.Staff), "access_nodes": len(r.AccessNodes), "buildings": len(b), "work_types": len(r.WorkTypes)}
}
func (s *Service) ReadReference(ctx context.Context) (json.RawMessage, error) {
	var b []byte
	e := s.Store.Pool.QueryRow(ctx, "SELECT document FROM reference_data WHERE id=true").Scan(&b)
	return b, e
}
