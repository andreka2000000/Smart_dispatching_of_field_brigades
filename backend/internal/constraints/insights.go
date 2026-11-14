package constraints

import (
	"magistral/backend/internal/dto"
	"reflect"
	"time"
)

// Insights checks audit invariants independently of incident detection.
func (w *World) Insights(p dto.Plan) error {
	now := Time(p.AsOf)
	groups := map[string]dto.PgpGroup{}
	alerts := map[string]dto.Alert{}
	for _, g := range p.PgpGroups {
		if _, ok := groups[g.GroupId]; ok || g.GroupId == "" || len(g.OrderIds) < 2 {
			return bad("INSIGHTS", "invalid PGP identity or membership")
		}
		groups[g.GroupId] = g
		if Time(g.DetectedAt).IsZero() || Time(g.DetectedAt).After(now) {
			return bad("INSIGHTS", "invalid PGP detection time")
		}
		var first, last time.Time
		for _, id := range g.OrderIds {
			o, ok := w.Orders[id]
			if !ok || o.AccessNodeId != g.AccessNodeId || !w.Work[o.WorkTypeCode].PgpRelevant {
				return bad("INSIGHTS", "PGP members must be relevant orders from the same node")
			}
			t := Time(o.RegisteredAt)
			if first.IsZero() || t.Before(first) {
				first = t
			}
			if last.IsZero() || t.After(last) {
				last = t
			}
		}
		if last.Sub(first) > time.Hour {
			return bad("INSIGHTS", "PGP registration range exceeds 60 minutes")
		}
	}
	for _, a := range p.Alerts {
		if _, ok := alerts[a.AlertId]; ok || a.AlertId == "" {
			return bad("INSIGHTS", "duplicate or empty alert ID")
		}
		alerts[a.AlertId] = a
		created := Time(a.CreatedAt)
		if created.IsZero() || created.After(now) {
			return bad("INSIGHTS", "invalid alert creation time")
		}
		if a.Status == "RESOLVED" {
			if a.ResolvedAt == nil || Time(*a.ResolvedAt).Before(created) || Time(*a.ResolvedAt).After(now) {
				return bad("INSIGHTS", "invalid alert resolution time")
			}
		} else if a.Status != "ACTIVE" || a.ResolvedAt != nil {
			return bad("INSIGHTS", "invalid alert status")
		}
	}
	if w.Previous != nil {
		oldIDs := map[string]bool{}
		for _, old := range w.Previous.Alerts {
			oldIDs[old.AlertId] = true
			a, ok := alerts[old.AlertId]
			if !ok {
				return bad("INSIGHTS", "alert history was removed")
			}
			if a.CreatedAt != old.CreatedAt || a.Type != old.Type || !reflect.DeepEqual(a.OrderId, old.OrderId) || !reflect.DeepEqual(a.GroupId, old.GroupId) || !reflect.DeepEqual(a.ForecastStart, old.ForecastStart) {
				return bad("INSIGHTS", "original alert evidence was changed")
			}
			if old.Status == "RESOLVED" && !reflect.DeepEqual(a, old) {
				return bad("INSIGHTS", "resolved alert was changed")
			}
		}
		for id, a := range alerts {
			if !oldIDs[id] && !Time(a.CreatedAt).Equal(now) {
				return bad("INSIGHTS", "new alert was backdated")
			}
		}
		for _, old := range w.Previous.PgpGroups {
			if g, ok := groups[old.GroupId]; ok && g.DetectedAt != old.DetectedAt {
				return bad("INSIGHTS", "PGP detection time was changed")
			}
		}
	}
	return nil
}
