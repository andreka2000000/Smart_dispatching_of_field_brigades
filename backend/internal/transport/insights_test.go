package transport

import (
	"context"
	"encoding/json"
	c "magistral/backend/internal/constraints"
	"magistral/backend/internal/dto"
	"os"
	"reflect"
	"testing"
	"time"
)

func pgpDay(t *testing.T, n int) dto.DayInput {
	t.Helper()
	b, e := os.ReadFile("../../testdata/day.json")
	if e != nil {
		t.Fatal(e)
	}
	var day dto.DayInput
	json.Unmarshal(b, &day)
	base := day.Orders[0]
	day.Orders = []dto.Order{}
	for i := 0; i < n; i++ {
		o := base
		o.OrderId = []string{"ORD-20261026-0081", "ORD-20261026-0082", "ORD-20261026-0083"}[i]
		o.RegisteredAt = c.Stamp(c.Time("2026-10-26T08:00:00+05:00").Add(time.Duration(i) * 10 * time.Minute))
		day.Orders = append(day.Orders, o)
	}
	return day
}
func findAlert(t *testing.T, p dto.Plan, kind, order string) dto.Alert {
	t.Helper()
	for _, a := range p.Alerts {
		if a.Type == kind && (order == "" || (a.OrderId != nil && *a.OrderId == order)) {
			return a
		}
	}
	t.Fatalf("missing %s alert for %s", kind, order)
	return dto.Alert{}
}
func TestPGPInitialAndDynamicHistory(t *testing.T) {
	f := eventFixture(t)
	day := pgpDay(t, 3)
	f.call("PUT", dayURL, encode(day), 200)
	f.build()
	p := f.plan()
	if len(p.PgpGroups) != 1 || *p.Summary.PgpOrders != 3 || *p.Summary.VisitsPlanned != 0 || len(p.Unassigned) != 0 {
		t.Fatal("initial PGP accounting failed")
	}
	group := p.PgpGroups[0]
	alert := findAlert(t, p, "PGP_DETECTED", "")
	if alert.CreatedAt != day.Now || !*alert.Applied {
		t.Fatal("PGP warning not applied at model now")
	}
	f.restart()
	f.event(tick("group-tick", "08:40:00"), 200)
	p = f.plan()
	if p.PgpGroups[0].GroupId != group.GroupId || p.PgpGroups[0].DetectedAt != group.DetectedAt || len(p.Alerts) != 1 {
		t.Fatal("restart recreated group or warning")
	}
	ev := f.newOrder("ORD-20261026-0084", "08:45:00")
	r := f.event(ev, 200)
	p = f.plan()
	if len(p.PgpGroups) != 1 || len(p.PgpGroups[0].OrderIds) != 4 || p.PgpGroups[0].GroupId != group.GroupId || len(r.NewAlerts) != 0 {
		t.Fatal("group extension lost identity")
	}
	cancel := tick("cancel-group-1", "08:50:00")
	cancel.Type = "ORDER_CANCELLED"
	cancel.Reason = "CUSTOMER_REQUEST"
	cancel.OrderId = day.Orders[0].OrderId
	f.event(cancel, 200)
	cancel.EventId = "cancel-group-2"
	cancel.OrderId = day.Orders[1].OrderId
	f.event(cancel, 200)
	p = f.plan()
	if len(p.PgpGroups) != 1 || len(p.PgpGroups[0].OrderIds) != 2 {
		t.Fatal("confirmed pair disappeared")
	}
	cancel.EventId = "cancel-group-3"
	cancel.OrderId = day.Orders[2].OrderId
	f.event(cancel, 200)
	p = f.plan()
	closed := findAlert(t, p, "PGP_DETECTED", "")
	if len(p.PgpGroups) != 0 || closed.Status != "RESOLVED" || closed.Applied == nil || !*closed.Applied {
		t.Fatal("singleton was not released or warning not closed")
	}
	f.restart()
	f.event(tick("closed-tick", "08:55:00"), 200)
	if !reflect.DeepEqual(closed, findAlert(t, f.plan(), "PGP_DETECTED", "")) {
		t.Fatal("closed warning changed after restart")
	}
}
func TestThirdEventPGPAndFrozenVisitExclusion(t *testing.T) {
	f := eventFixture(t)
	day := pgpDay(t, 2)
	f.call("PUT", dayURL, encode(day), 200)
	f.build()
	before := f.plan()
	ev := f.newOrder("ORD-20261026-0083", "08:50:00")
	r := f.event(ev, 200)
	p := f.plan()
	if len(p.PgpGroups) != 1 || *p.Summary.PgpOrders != 3 || *p.Summary.VisitsPlanned != 0 || len(r.NewAlerts) != 1 || r.NewAlerts[0].Type != "PGP_DETECTED" {
		t.Fatal("third event did not transfer the group")
	}
	if p.PlanVersion != before.PlanVersion+1 {
		t.Fatal("PGP added a separate version")
	}
	f = eventFixture(t)
	day = pgpDay(t, 2)
	day.Now = "2026-10-26T09:00:00+05:00"
	for i := range day.Orders {
		day.Orders[i].RegisteredAt = c.Stamp(c.Time("2026-10-26T08:50:00+05:00").Add(time.Duration(i) * 5 * time.Minute))
	}
	f.call("PUT", dayURL, encode(day), 200)
	f.build()
	old := f.plan()
	ev = f.newOrder("ORD-20261026-0083", "09:30:00")
	f.event(ev, 200)
	if len(f.plan().PgpGroups) != 0 {
		t.Fatal("started visit was counted toward threshold")
	}
	ev = f.newOrder("ORD-20261026-0084", "09:30:00")
	f.event(ev, 200)
	p = f.plan()
	if len(p.PgpGroups) != 1 || len(p.PgpGroups[0].OrderIds) != 3 || *p.Summary.VisitsPlanned != 1 {
		t.Fatal("future-only PGP incorrect")
	}
	for _, g := range p.PgpGroups {
		for _, r := range old.Routes {
			for _, s := range r.Stops {
				if s.OrderId != nil && c.Time(s.Start).Before(c.Time(p.AsOf)) && c.Has(g.OrderIds, *s.OrderId) {
					t.Fatal("started visit was grouped")
				}
			}
		}
	}
}
func TestRiskBeforeRepairAndAuditRollback(t *testing.T) {
	f := eventFixture(t)
	var ref dto.ReferenceData
	b, _ := os.ReadFile("../../testdata/reference.json")
	json.Unmarshal(b, &ref)
	staff := ref.Staff[0]
	staff.StaffId = "ST-002"
	staff.Shift.Start = "11:00"
	ref.Staff = append(ref.Staff, staff)
	f.call("PUT", "/api/v1/reference", encode(ref), 200)
	var day dto.DayInput
	b, _ = os.ReadFile("../../testdata/day.json")
	json.Unmarshal(b, &day)
	o := day.Orders[0]
	o.OrderId = "ORD-20261026-0043"
	o.Slot.Start = "2026-10-26T11:00:00+05:00"
	o.Slot.End = "2026-10-26T12:00:00+05:00"
	o.LatestStart = o.Slot.End
	o.Deadline = "2026-10-26T16:00:00+05:00"
	day.Orders = append(day.Orders, o)
	f.call("PUT", dayURL, encode(day), 200)
	f.build()
	ev := tick("risk-repair", "09:30:00")
	ev.Type = "VISIT_DELAYED"
	ev.StaffId = "ST-001"
	ev.OrderId = day.Orders[0].OrderId
	end := dto.DateTime("2026-10-26T12:10:00+05:00")
	ev.ExpectedEnd = &end
	// Snapshot failure must not publish a risk, group registry, or alert episode.
	before := f.plan()
	_, e := f.store.Pool.Exec(context.Background(), `CREATE FUNCTION reject_insight() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'injected insight failure'; END $$; CREATE TRIGGER reject_insight BEFORE INSERT ON events FOR EACH ROW EXECUTE FUNCTION reject_insight()`)
	if e != nil {
		t.Fatal(e)
	}
	f.event(ev, 500)
	if !reflect.DeepEqual(before, f.plan()) {
		t.Fatal("warning escaped rollback")
	}
	f.store.Pool.Exec(context.Background(), "DROP TRIGGER reject_insight ON events")
	result := f.event(ev, 200)
	p := f.plan()
	a := findAlert(t, p, "LATE_RISK", o.OrderId)
	if a.ForecastStart == nil || !c.Time(*a.ForecastStart).After(c.Time(o.LatestStart)) || a.LatenessMin == nil || *a.LatenessMin != 10 {
		t.Fatalf("lost pre-repair forecast: %+v", a)
	}
	if a.CreatedAt != ev.Timestamp || c.Time(a.CreatedAt).After(c.Time(o.Slot.Start).Add(-30*time.Minute)) || a.Status != "RESOLVED" || a.ResolvedAt == nil || !*a.Applied || a.SuggestedAction != "REASSIGN" || a.SuggestedStaffId == nil || *a.SuggestedStaffId != "ST-002" {
		t.Fatalf("risk audit/action incorrect: %+v", a)
	}
	if len(result.NewAlerts) != 1 || result.NewAlerts[0].AlertId != a.AlertId {
		t.Fatal("event result omitted new risk")
	}
	f.restart()
	f.event(tick("risk-later", "09:35:00"), 200)
	if !reflect.DeepEqual(a, findAlert(t, f.plan(), "LATE_RISK", o.OrderId)) {
		t.Fatal("closed evidence changed")
	}
	duplicate := f.event(ev, 200)
	if !duplicate.Duplicate || !reflect.DeepEqual(duplicate.NewAlerts, result.NewAlerts) {
		t.Fatal("duplicate did not retain original warnings")
	}
}
func TestUnassignedWarningsRemainActiveUntilResolved(t *testing.T) {
	f := eventFixture(t)
	f.build()
	ev := tick("risk-unavailable", "08:50:00")
	ev.Type = "STAFF_UNAVAILABLE"
	ev.StaffId = "ST-001"
	ev.Reason = "ILLNESS"
	result := f.event(ev, 200)
	p := f.plan()
	id := "ORD-20261026-0001"
	a := findAlert(t, p, "LATE_RISK", id)
	u := findAlert(t, p, "UNASSIGNED", id)
	if len(p.Unassigned) != 1 || p.Unassigned[0].ReasonCode != "STAFF_UNAVAILABLE" || a.Status != "ACTIVE" || *a.Applied || a.ForecastStart != nil || u.Status != "ACTIVE" || len(result.NewAlerts) != 2 {
		t.Fatal("unavailable warning or reason is misleading")
	}
	f.event(tick("risk-still-active", "08:55:00"), 200)
	if findAlert(t, f.plan(), "LATE_RISK", id).CreatedAt != a.CreatedAt || len(f.plan().Alerts) != 2 {
		t.Fatal("active episode was recreated")
	}
	cancel := tick("resolve-cancel", "08:55:00")
	cancel.Type = "ORDER_CANCELLED"
	cancel.OrderId = id
	cancel.Reason = "NOT_CONFIRMED"
	f.event(cancel, 200)
	closed := findAlert(t, f.plan(), "LATE_RISK", id)
	if closed.Status != "RESOLVED" || *closed.Applied {
		t.Fatal("cancellation falsely applied a reschedule")
	}
	f.event(tick("resolve-later", "09:00:00"), 200)
	if !reflect.DeepEqual(closed, findAlert(t, f.plan(), "LATE_RISK", id)) {
		t.Fatal("resolved timestamp was rewritten")
	}
}
