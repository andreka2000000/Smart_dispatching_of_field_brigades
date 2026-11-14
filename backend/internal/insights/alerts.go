package insights

import (
	"fmt"
	c "magistral/backend/internal/constraints"
	"magistral/backend/internal/dto"
	"math"
	"sort"
)

// Alerts never removes history or changes a resolved alert. An active episode
// retains its first observed forecast and created_at; a recurrence gets a new ID.
func Alerts(w *c.World, old []dto.Alert, risks []Risk, p *dto.Plan) []dto.Alert {
	p.Alerts = append([]dto.Alert{}, old...)
	known := map[string]bool{}
	active := map[string]int{}
	episodes := map[string]int{}
	key := func(a dto.Alert) string {
		x := ""
		if a.OrderId != nil {
			x = *a.OrderId
		}
		if a.GroupId != nil {
			x = *a.GroupId
		}
		return a.Type + "|" + x
	}
	for i, a := range old {
		k := key(a)
		known[a.AlertId] = true
		episodes[k]++
		if a.Status == "ACTIVE" {
			active[k] = i
		}
	}
	add := func(a dto.Alert) int {
		k := key(a)
		if i, ok := active[k]; ok {
			return i
		}
		episodes[k]++
		a.AlertId = id("ALR", w.Day.Date+"|"+k+fmt.Sprint("|", episodes[k]))
		a.CreatedAt = w.Day.Now
		a.Status = "ACTIVE"
		a.Applied = ptr(false)
		p.Alerts = append(p.Alerts, a)
		i := len(p.Alerts) - 1
		active[k] = i
		return i
	}
	assigned := map[string]string{}
	for _, r := range p.Routes {
		for _, s := range r.Stops {
			if s.Type == "VISIT" {
				assigned[*s.OrderId] = r.StaffId
			}
		}
	}
	grouped := map[string]string{}
	groups := map[string]bool{}
	for _, g := range p.PgpGroups {
		groups[g.GroupId] = true
		for _, x := range g.OrderIds {
			grouped[x] = g.GroupId
		}
	}
	cancelled := map[string]bool{}
	for _, x := range p.Cancelled {
		cancelled[x.OrderId] = true
	}
	unassigned := map[string]bool{}
	for _, u := range p.Unassigned {
		unassigned[u.OrderId] = true
	}
	resolve := func(i int, applied bool) {
		a := &p.Alerts[i]
		a.Status = "RESOLVED"
		a.ResolvedAt = ptr(w.Day.Now)
		a.Applied = ptr(applied)
	}
	for _, g := range p.PgpGroups {
		i := add(dto.Alert{Type: "PGP_DETECTED", GroupId: ptr(g.GroupId), SuggestedAction: "TRANSFER_TO_STATION", Message: fmt.Sprintf("ПГП на %s: %d нарядов требуют станционной диагностики.", g.AccessNodeId, len(g.OrderIds))})
		p.Alerts[i].Applied = ptr(true)
		p.Alerts[i].Message = fmt.Sprintf("ПГП на %s: %d нарядов требуют станционной диагностики.", g.AccessNodeId, len(g.OrderIds))
	}
	sort.Slice(risks, func(i, j int) bool { return risks[i].OrderID < risks[j].OrderID })
	for _, risk := range risks {
		if grouped[risk.OrderID] != "" || cancelled[risk.OrderID] {
			continue
		}
		a := dto.Alert{Type: "LATE_RISK", OrderId: ptr(risk.OrderID), StaffId: ptr(risk.StaffID), ForecastStart: risk.Forecast, SuggestedAction: "RESCHEDULE", Message: "Исходный маршрут не позволяет начать визит в допустимое время; требуется вмешательство."}
		if risk.Forecast != nil {
			late := int64(math.Ceil(c.Time(*risk.Forecast).Sub(c.Time(w.Orders[risk.OrderID].LatestStart)).Minutes()))
			a.LatenessMin = &late
		} else {
			a.Message = "Исходный маршрут не позволяет выполнить визит; требуется замена исполнителя или согласование переноса."
		}
		i := add(a)
		if staff, ok := assigned[risk.OrderID]; ok {
			a := &p.Alerts[i]
			if staff != risk.StaffID {
				a.SuggestedAction = "REASSIGN"
				a.SuggestedStaffId = ptr(staff)
				a.Message = "Риск опоздания устранён: наряд передан другому сотруднику."
			} else {
				a.SuggestedAction = "NONE"
				a.Message = "Риск опоздания устранён перестроением маршрута."
			}
			resolve(i, true)
		} else if risk.Forecast != nil {
			// A proposal is informational. The agreed slot is never silently modified.
			order := w.Orders[risk.OrderID]
			start := c.Time(*risk.Forecast)
			width := c.Time(order.Slot.End).Sub(c.Time(order.Slot.Start))
			fits := false
			for _, s := range w.Reference.Staff {
				if c.Qualified(s, order, w.Work[order.WorkTypeCode]) && !start.Add(width).After(c.Local(w.Day.Date, s.Shift.End)) {
					fits = true
				}
			}
			if width > 0 && start.Format("2006-01-02") == w.Day.Date && fits {
				p.Alerts[i].ProposedSlot = &dto.TimeWindow{Start: c.Stamp(start), End: c.Stamp(start.Add(width))}
			}
		}
	}
	for _, u := range p.Unassigned {
		action := "RESCHEDULE"
		if u.ReasonCode == "NO_QUALIFIED_STAFF" {
			action = "REASSIGN"
		}
		i := add(dto.Alert{Type: "UNASSIGNED", OrderId: ptr(u.OrderId), SuggestedAction: action, Message: u.ReasonText})
		p.Alerts[i].Message = u.ReasonText
		p.Alerts[i].SuggestedAction = action
	}
	// Resolve active episodes only. Their evidence remains in earlier snapshots;
	// closed episodes are immutable and keep their original resolution time.
	for i, a := range p.Alerts {
		if a.Status != "ACTIVE" {
			continue
		}
		switch a.Type {
		case "PGP_DETECTED":
			if a.GroupId != nil && !groups[*a.GroupId] {
				resolve(i, a.Applied != nil && *a.Applied)
			}
		case "UNASSIGNED":
			if a.OrderId != nil && !unassigned[*a.OrderId] {
				if grouped[*a.OrderId] != "" {
					p.Alerts[i].SuggestedAction = "TRANSFER_TO_STATION"
				} else if staff, ok := assigned[*a.OrderId]; ok {
					p.Alerts[i].SuggestedAction = "REASSIGN"
					p.Alerts[i].SuggestedStaffId = ptr(staff)
				} else {
					p.Alerts[i].SuggestedAction = "NONE"
				}
				resolve(i, !cancelled[*a.OrderId])
			}
		case "LATE_RISK":
			if a.OrderId == nil {
				continue
			}
			order := *a.OrderId
			if cancelled[order] || grouped[order] != "" {
				resolve(i, grouped[order] != "")
			} else if staff, ok := assigned[order]; ok {
				p.Alerts[i].SuggestedStaffId = ptr(staff)
				if a.StaffId != nil && *a.StaffId != staff {
					p.Alerts[i].SuggestedAction = "REASSIGN"
				} else {
					p.Alerts[i].SuggestedAction = "NONE"
				}
				resolve(i, true)
			}
		}
	}
	fresh := []dto.Alert{}
	for _, a := range p.Alerts {
		if !known[a.AlertId] {
			fresh = append(fresh, a)
		}
	}
	return fresh
}
