package insights

import (
	c "magistral/backend/internal/constraints"
	"magistral/backend/internal/dto"
	"time"
)

type Risk struct {
	OrderID, StaffID string
	Forecast         *dto.DateTime
	Blocked          bool
}

func max(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

// Forecast follows the previous route without repair, relaxing latest_start so
// that lateness remains visible. It must run before scheduling moves the job.
// Unchanged legs may already be in transit; they are not restarted from now.
func Forecast(w *c.World, old dto.Plan, cancelled map[string]dto.CancelledOrder) []Risk {
	excluded := map[string]bool{}
	for x := range cancelled {
		excluded[x] = true
	}
	for _, g := range w.Groups {
		for _, x := range g.OrderIds {
			excluded[x] = true
		}
	}
	risks := []Risk{}
	now := c.Time(w.Day.Now)
	for _, r := range old.Routes {
		var cursor time.Time
		loc := r.ServiceCenterId
		blocked := false
		for _, s := range r.Stops {
			if c.Time(s.Start).Before(now) {
				cursor = c.Time(s.End)
				if s.OrderId != nil {
					if end, ok := w.EndOverrides[*s.OrderId]; ok {
						cursor = c.Time(end)
					}
				}
				loc = s.LocationId
				continue
			}
			if s.OrderId != nil && excluded[*s.OrderId] {
				continue
			}
			ready := cursor
			for _, u := range w.Unavailable[r.StaffId] {
				if u.Until == nil {
					blocked = true
					continue
				}
				until := c.Time(*u.Until)
				if until.After(ready) {
					ready = until
					loc = r.ServiceCenterId
				}
			}
			travel := int64(0)
			if s.Type != "BREAK" {
				travel, _, _ = w.Matrix.Travel(loc, s.LocationId)
			}
			arrival := max(c.Time(s.Arrival), ready.Add(time.Duration(travel)*time.Second))
			start := max(arrival, c.Time(s.Start))
			end := start
			switch s.Type {
			case "VISIT":
				o := w.Orders[*s.OrderId]
				start = max(start, c.Time(o.Slot.Start))
				end = start.Add(time.Duration(w.Work[o.WorkTypeCode].DurationMin) * time.Minute)
				if override, ok := w.EndOverrides[o.OrderId]; ok {
					end = c.Time(override)
				}
				if !excluded[o.OrderId] && (blocked || start.After(c.Time(o.LatestStart))) {
					risk := Risk{OrderID: o.OrderId, StaffID: r.StaffId, Blocked: blocked}
					if !blocked {
						risk.Forecast = ptr(c.Stamp(start))
					}
					risks = append(risks, risk)
				}
			case "BREAK":
				start = max(start, c.Local(w.Day.Date, w.Staff[r.StaffId].Shift.BreakWindow.From))
				end = start.Add(time.Duration(w.Staff[r.StaffId].Shift.BreakDurationMin) * time.Minute)
			case "RESTOCK":
				center := w.Centers[s.LocationId]
				start = max(start, c.Local(w.Day.Date, center.OpeningHours.Open))
				end = start.Add(time.Duration(center.RestockDurationMin) * time.Minute)
				if end.After(c.Local(w.Day.Date, center.OpeningHours.Close)) {
					blocked = true
				}
			}
			cursor = end
			loc = s.LocationId
		}
	}
	return risks
}
