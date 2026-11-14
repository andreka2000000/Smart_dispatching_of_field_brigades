package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"magistral/backend/internal/application"
	c "magistral/backend/internal/constraints"
	"magistral/backend/internal/contract"
	"magistral/backend/internal/dto"
	"magistral/backend/internal/storage/postgres"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

type eventsFixture struct {
	t         *testing.T
	store     *postgres.Store
	api       http.Handler
	validator *contract.Validator
	url       string
}

const dayURL = "/api/v1/days/2026-10-26"

func eventFixture(t *testing.T) *eventsFixture {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set TEST_DATABASE_URL for event integration tests")
	}
	ctx := context.Background()
	root, e := postgres.Open(ctx, url)
	if e != nil {
		t.Fatal(e)
	}
	schema := fmt.Sprintf("events_test_%d", time.Now().UnixNano())
	if _, e = root.Pool.Exec(ctx, "CREATE SCHEMA "+schema); e != nil {
		t.Fatal(e)
	}
	sep := "?"
	if strings.Contains(url, "?") {
		sep = "&"
	}
	url += sep + "search_path=" + schema
	store, e := postgres.Open(ctx, url)
	if e != nil {
		t.Fatal(e)
	}
	v, e := contract.New()
	if e != nil {
		t.Fatal(e)
	}
	f := &eventsFixture{t: t, store: store, validator: v, url: url}
	f.api = (&API{Service: &application.Service{Store: store}, Validator: v}).Handler()
	t.Cleanup(func() { store.Pool.Close(); root.Pool.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE"); root.Pool.Close() })
	for _, item := range []struct{ name, path string }{{"reference", "/api/v1/reference"}, {"matrix", "/api/v1/reference/travel-matrix"}, {"day", dayURL}} {
		b, e := os.ReadFile("../../testdata/" + item.name + ".json")
		if e != nil {
			t.Fatal(e)
		}
		f.call("PUT", item.path, b, 200)
	}
	return f
}
func (f *eventsFixture) call(method, path string, body []byte, status int) []byte {
	f.t.Helper()
	rr := httptest.NewRecorder()
	f.api.ServeHTTP(rr, httptest.NewRequest(method, path, bytes.NewReader(body)))
	if rr.Code != status {
		f.t.Fatalf("%s %s: expected %d, got %d: %s", method, path, status, rr.Code, rr.Body.String())
	}
	if status >= 400 {
		var v dto.ErrorResponse
		if e := json.Unmarshal(rr.Body.Bytes(), &v); e != nil || v.Error.Code == "" {
			f.t.Fatal("bad error envelope")
		}
	}
	return rr.Body.Bytes()
}
func (f *eventsFixture) plan() dto.Plan {
	f.t.Helper()
	var p dto.Plan
	b := f.call("GET", dayURL+"/plan", nil, 200)
	if e := f.validator.Decode(b, "Plan", &p); e != nil {
		f.t.Fatal(e)
	}
	return p
}
func (f *eventsFixture) build() { f.call("POST", dayURL+"/plan", nil, 200) }
func tick(id, at string) dto.EventInput {
	return dto.EventInput{EventId: id, Type: "TIME_TICK", Timestamp: dto.DateTime("2026-10-26T" + at + "+05:00")}
}
func encode(v any) []byte {
	b, e := json.Marshal(v)
	if e != nil {
		panic(e)
	}
	return b
}
func (f *eventsFixture) event(ev dto.EventInput, status int) dto.EventResult {
	f.t.Helper()
	var before dto.Plan
	if _, e := f.store.CurrentPlan(context.Background(), "2026-10-26"); e == nil {
		before = f.plan()
	}
	b := f.call("POST", dayURL+"/events", encode(ev), status)
	var result dto.EventResult
	if status == 200 {
		if e := f.validator.Decode(b, "EventResult", &result); e != nil {
			f.t.Fatal(e)
		}
		p := f.plan()
		if !result.Duplicate {
			if p.PlanVersion != result.PlanVersion || p.BasedOnEventId == nil || *p.BasedOnEventId != ev.EventId {
				f.t.Fatal("event and plan are not atomic")
			}
			in, e := f.store.EventInputs(context.Background(), "2026-10-26")
			if e != nil {
				f.t.Fatal(e)
			}
			w, e := c.New(in.Reference, in.Day, in.Matrix)
			if e != nil {
				f.t.Fatal(e)
			}
			w.Previous = &before
			w.EndOverrides = in.State.EndOverrides
			w.NewOrderAt = in.State.NewOrderAt
			w.Unavailable = in.State.Unavailable
			if e = w.Validate(p); e != nil {
				f.t.Fatal("event plan rejected", e)
			}
		}
	} else if before.PlanVersion > 0 && !reflect.DeepEqual(before, f.plan()) {
		f.t.Fatal("failed event changed plan")
	}
	return result
}
func (f *eventsFixture) restart() {
	f.t.Helper()
	store, e := postgres.Open(context.Background(), f.url)
	if e != nil {
		f.t.Fatal(e)
	}
	f.t.Cleanup(store.Pool.Close)
	f.api = (&API{Service: &application.Service{Store: store}, Validator: f.validator}).Handler()
}
func (f *eventsFixture) newOrder(id, at string) dto.EventInput {
	f.t.Helper()
	b, _ := os.ReadFile("../../testdata/day.json")
	var d dto.DayInput
	json.Unmarshal(b, &d)
	o := d.Orders[0]
	o.OrderId = id
	o.RegisteredAt = dto.DateTime("2026-10-26T" + at + "+05:00")
	o.Slot.Start = "2026-10-26T14:00:00+05:00"
	o.Slot.End = "2026-10-26T16:00:00+05:00"
	o.LatestStart = o.Slot.End
	o.Deadline = "2026-10-26T18:00:00+05:00"
	ev := tick("new-"+id, at)
	ev.Type = "NEW_ORDER"
	ev.Order = &o
	return ev
}

func TestEventLifecycle(t *testing.T) {
	f := eventFixture(t)
	f.event(tick("before-plan", "08:35:00"), 409)
	f.call("POST", "/api/v1/days/2099-01-01/events", encode(tick("missing-day", "08:40:00")), 404)
	f.build()
	first := f.event(tick("a", "08:45:00"), 200)
	if first.PlanVersion != 2 || first.Duplicate {
		t.Fatal(first)
	}
	f.event(tick("b", "08:50:00"), 200)
	f.event(tick("equal", "08:50:00"), 200)
	p := f.plan()
	dup := f.event(tick("a", "08:45:00"), 200)
	if !dup.Duplicate || dup.PlanVersion != 2 || dup.AsOf != first.AsOf || !reflect.DeepEqual(p, f.plan()) {
		t.Fatal("duplicate advanced state")
	}
	f.event(tick("old", "08:49:59"), 409)
	f.call("POST", dayURL+"/plan", nil, 409)
	ev := f.newOrder("ORD-20261026-0099", "08:50:00")
	f.event(ev, 200)
	if *f.plan().Summary.OrdersTotal != 3 {
		t.Fatal("new order not persisted")
	}
	f.event(ev, 200)
	cancel := tick("cancel", "08:50:00")
	cancel.Type = "ORDER_CANCELLED"
	cancel.OrderId = ev.Order.OrderId
	cancel.Reason = "CUSTOMER_REQUEST"
	f.event(cancel, 200)
	if len(f.plan().Cancelled) != 1 {
		t.Fatal("cancellation missing")
	}
	cancel.EventId = "cancel-again"
	f.event(cancel, 422)
	unknown := cancel
	unknown.EventId = "unknown"
	unknown.OrderId = "missing"
	f.event(unknown, 422)
	invalid := f.newOrder("ORD-20261026-0098", "08:50:00")
	invalid.Order.BuildingId = "B-99999"
	f.event(invalid, 422)
	f.event(tick("", "08:50:00"), 400)
	f.call("POST", dayURL+"/events", []byte(`{"event_id":"bad","type":"NEW_ORDER","timestamp":"2026-10-26T08:50:00+05:00"}`), 400)
	f.restart()
	dup = f.event(tick("a", "08:45:00"), 200)
	if !dup.Duplicate || dup.PlanVersion != 2 {
		t.Fatal("restart lost dedup")
	}
	var journal []dto.EventInput
	json.Unmarshal(f.call("GET", "/ui/v1/days/2026-10-26/events", nil, 200), &journal)
	if len(journal) != 5 {
		t.Fatalf("journal includes rejected or duplicate events: %d", len(journal))
	}
}
func TestDelayTransferAndStartedCancel(t *testing.T) {
	f := eventFixture(t)
	f.build()
	old := f.plan()
	var target dto.Stop
	var staff string
	for _, r := range old.Routes {
		for _, x := range r.Stops {
			if x.Type == "VISIT" && *x.OrderId == "ORD-20261026-0001" {
				target = x
				staff = r.StaffId
			}
		}
	}
	f.event(tick("advance", "09:20:00"), 200)
	delayed := tick("delay", "09:30:00")
	delayed.Type = "VISIT_DELAYED"
	delayed.OrderId = *target.OrderId
	delayed.StaffId = staff
	end := dto.DateTime("2026-10-26T11:10:00+05:00")
	delayed.ExpectedEnd = &end
	f.event(delayed, 200)
	f.restart()
	f.event(tick("after-restart", "09:35:00"), 200)
	transferred := tick("transfer", "09:40:00")
	transferred.Type = "ORDER_TRANSFERRED"
	transferred.OrderId = *target.OrderId
	transferred.StaffId = staff
	end = "2026-10-26T09:40:00+05:00"
	transferred.ActualEnd = &end
	transferred.TransferredTo = "STATION_EQUIPMENT"
	f.event(transferred, 200)
	p := f.plan()
	for _, r := range p.Routes {
		for _, x := range r.Stops {
			if x.Type == "VISIT" && *x.OrderId == *target.OrderId {
				if x.Arrival != target.Arrival || x.Start != target.Start || x.End != end {
					t.Fatal("frozen visit changed")
				}
			}
		}
	}
	transferred.EventId = "again"
	f.event(transferred, 422)
	cancel := tick("cancel-started", "09:45:00")
	cancel.Type = "ORDER_CANCELLED"
	cancel.OrderId = *target.OrderId
	cancel.Reason = "CUSTOMER_REQUEST"
	f.event(cancel, 200)
	if len(f.plan().Cancelled) != 0 {
		t.Fatal("started visit cancelled")
	}
	// Backward or contradictory end changes leave every persisted field intact.
	delayed.EventId = "bad-staff"
	delayed.Timestamp = cancel.Timestamp
	delayed.StaffId = "ST-999"
	f.event(delayed, 422)
}
func TestUnavailableAndRecovery(t *testing.T) {
	for _, finite := range []bool{false, true} {
		t.Run(fmt.Sprint(finite), func(t *testing.T) {
			f := eventFixture(t)
			f.build()
			ev := tick("unavailable", "08:50:00")
			ev.Type = "STAFF_UNAVAILABLE"
			ev.StaffId = "ST-001"
			ev.Reason = "VEHICLE_BREAKDOWN"
			until := dto.DateTime("2026-10-26T10:00:00+05:00")
			if finite {
				ev.Until = &until
			}
			f.event(ev, 200)
			p := f.plan()
			if !finite && len(p.Unassigned) != 1 {
				t.Fatal("future visit of unavailable employee retained")
			}
			if finite {
				f.event(tick("resume", "10:10:00"), 200)
				if f.plan().Routes[0].Status != "ACTIVE" {
					t.Fatal("employee did not recover")
				}
			}
			f.restart()
			f.event(tick("later", "10:15:00"), 200)
		})
	}
}
func TestContradictoryHistoryRejected(t *testing.T) {
	f := eventFixture(t)
	f.build()
	ev := tick("outage-before-lunch", "09:30:00")
	ev.Type = "STAFF_UNAVAILABLE"
	ev.StaffId = "ST-001"
	ev.Reason = "ILLNESS"
	f.event(ev, 422) // no legal future BREAK exists during permanent unavailability
	ev.EventId = "unknown-staff"
	ev.StaffId = "ST-999"
	f.event(ev, 422)
}
func TestConcurrentEventsAndRollback(t *testing.T) {
	f := eventFixture(t)
	f.build()
	// Independent API instances and a small connection pool must not deadlock
	// when concurrent callers wait for the same day's PostgreSQL lock.
	otherAPI := f.api
	small, e := postgres.Open(context.Background(), f.url+"&pool_max_conns=2")
	if e != nil {
		t.Fatal(e)
	}
	defer small.Pool.Close()
	f.api = (&API{Service: &application.Service{Store: small}, Validator: f.validator}).Handler()
	var wg sync.WaitGroup
	responses := make(chan *httptest.ResponseRecorder, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		handler := f.api
		if i%2 == 0 {
			handler = otherAPI
		}
		go func(api http.Handler) {
			defer wg.Done()
			rr := httptest.NewRecorder()
			api.ServeHTTP(rr, httptest.NewRequest("POST", dayURL+"/events", bytes.NewReader(encode(tick("same", "08:40:00")))))
			responses <- rr
		}(handler)
	}
	wg.Wait()
	close(responses)
	accepted := 0
	for rr := range responses {
		var r dto.EventResult
		json.Unmarshal(rr.Body.Bytes(), &r)
		if rr.Code != 200 || r.PlanVersion != 2 {
			t.Fatal(rr.Code, rr.Body.String())
		}
		if !r.Duplicate {
			accepted++
		}
	}
	if accepted != 1 {
		t.Fatal("event accepted multiple times")
	}
	responses = make(chan *httptest.ResponseRecorder, 6)
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			rr := httptest.NewRecorder()
			f.api.ServeHTTP(rr, httptest.NewRequest("POST", dayURL+"/events", bytes.NewReader(encode(tick(fmt.Sprintf("equal-%d", i), "08:40:00")))))
			responses <- rr
		}(i)
	}
	wg.Wait()
	close(responses)
	versions := map[int64]bool{}
	for rr := range responses {
		var r dto.EventResult
		json.Unmarshal(rr.Body.Bytes(), &r)
		if rr.Code != 200 || r.Duplicate || r.PlanVersion < 3 || r.PlanVersion > 8 || versions[r.PlanVersion] {
			t.Fatal("concurrent versions lost", rr.Code, rr.Body.String())
		}
		versions[r.PlanVersion] = true
	}
	committed := f.plan()
	_, e = f.store.Pool.Exec(context.Background(), `CREATE FUNCTION reject_event() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'injected event failure'; END $$; CREATE TRIGGER reject_event BEFORE INSERT ON events FOR EACH ROW EXECUTE FUNCTION reject_event()`)
	if e != nil {
		t.Fatal(e)
	}
	ev := f.newOrder("ORD-20261026-0097", "08:45:00")
	f.event(ev, 500)
	if !reflect.DeepEqual(committed, f.plan()) {
		t.Fatal("rollback failed")
	}
	var orders int
	if e = f.store.Pool.QueryRow(context.Background(), "SELECT count(*) FROM orders").Scan(&orders); e != nil || orders != 2 {
		t.Fatal("new order escaped rollback", e)
	}
	if _, e = f.store.Pool.Exec(context.Background(), "DROP TRIGGER reject_event ON events"); e != nil {
		t.Fatal(e)
	}
	f.event(ev, 200)
	in, e := f.store.EventInputs(context.Background(), "2026-10-26")
	if e != nil {
		t.Fatal(e)
	}
	b, _ := os.ReadFile("../../testdata/day.json")
	f.call("PUT", dayURL, b, 200)
	if e = f.store.PublishEvent(context.Background(), in, tick("stale", "08:50:00"), in.Day, in.State, in.Plan, dto.EventResult{}); !errors.Is(e, postgres.ErrStale) {
		t.Fatal("stale event overwrote reset", e)
	}
	var count int
	if e = f.store.Pool.QueryRow(context.Background(), "SELECT count(*) FROM events").Scan(&count); e != nil || count != 0 {
		t.Fatal("reset retained journal", e)
	}
	f.build()
	r := f.event(tick("same", "08:40:00"), 200)
	if r.Duplicate {
		t.Fatal("new generation inherited deduplication")
	}
}

func TestEventQueueCancellationAndFinalCommitRollback(t *testing.T) {
	f := eventFixture(t)
	f.build()
	held, release, finished := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	go func() {
		finished <- f.store.WithDayLock(context.Background(), "2026-10-26", func() error { close(held); <-release; return nil })
	}()
	<-held
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	rr := httptest.NewRecorder()
	f.api.ServeHTTP(rr, httptest.NewRequest("POST", dayURL+"/events", bytes.NewReader(encode(tick("retry", "08:45:00")))).WithContext(ctx))
	close(release)
	if e := <-finished; e != nil {
		t.Fatal(e)
	}
	if rr.Code != 409 || f.plan().PlanVersion != 1 {
		t.Fatal("cancelled queue changed state", rr.Code)
	}
	f.event(tick("retry", "08:45:00"), 200)
	before := f.plan()
	journalBefore := f.call("GET", "/ui/v1/days/2026-10-26/events", nil, 200)
	_, e := f.store.Pool.Exec(context.Background(), `CREATE FUNCTION reject_day_commit() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.event_count > OLD.event_count THEN RAISE EXCEPTION 'injected final update failure'; END IF; RETURN NEW; END $$; CREATE TRIGGER reject_day_commit BEFORE UPDATE ON days FOR EACH ROW EXECUTE FUNCTION reject_day_commit()`)
	if e != nil {
		t.Fatal(e)
	}
	f.event(tick("commit-retry", "08:50:00"), 500)
	if !reflect.DeepEqual(before, f.plan()) || !bytes.Equal(journalBefore, f.call("GET", "/ui/v1/days/2026-10-26/events", nil, 200)) {
		t.Fatal("failed final update retained event or snapshot")
	}
	if _, e = f.store.Pool.Exec(context.Background(), "DROP TRIGGER reject_day_commit ON days"); e != nil {
		t.Fatal(e)
	}
	f.event(tick("commit-retry", "08:50:00"), 200)
}

func TestEventScaleBudget(t *testing.T) {
	f := eventFixture(t)
	var ref dto.ReferenceData
	var day dto.DayInput
	b, _ := os.ReadFile("../../testdata/reference.json")
	json.Unmarshal(b, &ref)
	// Exercise route repair, rather than grouping 400 identical synthetic repairs.
	for i := range ref.WorkTypes {
		ref.WorkTypes[i].PgpRelevant = false
	}
	b, _ = os.ReadFile("../../testdata/day.json")
	json.Unmarshal(b, &day)
	staff := ref.Staff[0]
	staff.Profiles = []dto.Profile{"INSTALL", "LTP3"}
	staff.EquipmentStart = dto.Equipment{ONT: 50, ROUTER: 50, STB: 50}
	staff.EquipmentCapacity = staff.EquipmentStart
	ref.Staff = []dto.Staff{}
	for i := 1; i <= 40; i++ {
		staff.StaffId = fmt.Sprintf("ST-%03d", i)
		ref.Staff = append(ref.Staff, staff)
	}
	order := day.Orders[0]
	day.Orders = []dto.Order{}
	order.Slot.End = "2026-10-26T17:00:00+05:00"
	order.LatestStart = order.Slot.End
	order.Deadline = "2026-10-26T18:00:00+05:00"
	for i := 1; i <= 400; i++ {
		order.OrderId = fmt.Sprintf("ORD-20261026-%04d", i)
		day.Orders = append(day.Orders, order)
	}
	f.call("PUT", "/api/v1/reference", encode(ref), 200)
	f.call("PUT", dayURL, encode(day), 200)
	f.call("POST", dayURL+"/plan", []byte(`{"time_limit_sec":1}`), 200)
	started := time.Now()
	f.event(tick("scale-tick", "08:31:00"), 200)
	if time.Since(started) > 10*time.Second {
		t.Fatal("event exceeded 10 seconds")
	}
	p := f.plan()
	if len(p.Routes) != 40 || *p.Summary.VisitsPlanned+int64(len(p.Unassigned)) != 400 {
		t.Fatal("scale event lost orders")
	}
	t.Logf("40 staff / 400 orders: event and checks took %v", time.Since(started))
}

func TestDelayedTailReassigned(t *testing.T) {
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
	order := day.Orders[0]
	order.OrderId = "ORD-20261026-0043"
	order.Slot.Start = "2026-10-26T11:00:00+05:00"
	order.Slot.End = "2026-10-26T12:00:00+05:00"
	order.LatestStart = order.Slot.End
	order.Deadline = "2026-10-26T16:00:00+05:00"
	day.Orders = append(day.Orders, order)
	f.call("PUT", dayURL, encode(day), 200)
	f.build()
	old := f.plan()
	owner := func(p dto.Plan, id string) string {
		for _, r := range p.Routes {
			for _, s := range r.Stops {
				if s.OrderId != nil && *s.OrderId == id {
					return r.StaffId
				}
			}
		}
		return ""
	}
	if owner(old, order.OrderId) != "ST-001" {
		t.Fatal("test setup did not place the future visit on ST-001")
	}
	ev := tick("delayed-tail", "09:30:00")
	ev.Type = "VISIT_DELAYED"
	ev.OrderId = "ORD-20261026-0001"
	ev.StaffId = "ST-001"
	end := dto.DateTime("2026-10-26T12:10:00+05:00")
	ev.ExpectedEnd = &end
	f.event(ev, 200)
	p := f.plan()
	if owner(p, order.OrderId) != "ST-002" {
		t.Fatal("affected future visit was not reassigned")
	}
	var a, before dto.Route
	for _, r := range old.Routes {
		if r.StaffId == "ST-029" {
			before = r
		}
	}
	for _, r := range p.Routes {
		if r.StaffId == "ST-029" {
			a = r
		}
	}
	if !reflect.DeepEqual(before.Stops, a.Stops) {
		t.Fatal("unaffected route changed")
	}
	f.restart()
	f.event(tick("after-delay-restart", "09:35:00"), 200)
}
func TestFiniteOutageAfterVisitAndCancellationBoundary(t *testing.T) {
	f := eventFixture(t)
	f.build()
	ev := tick("finite-after-start", "09:30:00")
	ev.Type = "STAFF_UNAVAILABLE"
	ev.StaffId = "ST-001"
	ev.Reason = "VEHICLE_BREAKDOWN"
	until := dto.DateTime("2026-10-26T11:30:00+05:00")
	ev.Until = &until
	f.event(ev, 200)
	home := false
	for _, r := range f.plan().Routes {
		if r.StaffId == "ST-001" {
			for _, s := range r.Stops {
				if s.Type == "RESTOCK" && s.LocationId == "SC-01" && !c.Time(s.Start).Before(c.Time(until)) {
					home = true
				}
			}
		}
	}
	if !home {
		t.Fatal("finite outage did not resume from home center")
	}
	f.event(tick("resume-prefix", "11:35:00"), 200)
	f = eventFixture(t)
	f.build()
	ev = tick("cancel-at-start", "09:18:00")
	ev.Type = "ORDER_CANCELLED"
	ev.OrderId = "ORD-20261026-0001"
	ev.Reason = "NOT_CONFIRMED"
	f.event(ev, 200)
	if len(f.plan().Cancelled) != 1 {
		t.Fatal("start == timestamp should remain mutable")
	}
}
