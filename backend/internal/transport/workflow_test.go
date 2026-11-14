package transport

import (
	"context"
	"encoding/json"
	"magistral/backend/internal/application"
	"magistral/backend/internal/dto"
	"sync"
	"testing"
	"time"
)

func newRequest(c *accessCase) string {
	raw := c.call("subscriber", "POST", "/ui/v1/requests", map[string]string{"service": "REPAIR", "address": "Учебный адрес абонента 1", "phone": "+79990000000", "description": "Не работает интернет"}, 201)
	var result map[string]string
	json.Unmarshal(raw.Body.Bytes(), &result)
	return result["id"]
}
func approvalInput() application.Approval {
	return application.Approval{Date: "2026-10-26", BuildingID: "B-00412", NodeID: "AN-0031", WorkType: "REPAIR", Slot: dto.TimeWindow{Start: "2026-10-26T11:00:00+05:00", End: "2026-10-26T13:00:00+05:00"}, Note: "Визит согласован", Equipment: dto.Equipment{}}
}
func TestSubscriberApprovalToConfirmedCompletion(t *testing.T) {
	c := accessSetup(t)
	id := newRequest(c)
	path := "/ui/v1/requests/" + id + "/approve"
	input := approvalInput()
	c.call("subscriber", "POST", path, input, 403)
	var first, duplicate dto.EventResult
	json.Unmarshal(c.call("dispatcher", "POST", path, input, 200).Body.Bytes(), &first)
	json.Unmarshal(c.call("dispatcher", "POST", path, input, 200).Body.Bytes(), &duplicate)
	if !duplicate.Duplicate || first.PlanVersion != duplicate.PlanVersion {
		t.Fatal("approval created duplicate event")
	}
	var requests []ServiceRequest
	json.Unmarshal(c.call("subscriber", "GET", "/ui/v1/requests", nil, 200).Body.Bytes(), &requests)
	if len(requests) != 1 || requests[0].Status != "APPROVED" || requests[0].Order == nil || requests[0].Order.State != "SCHEDULED" {
		t.Fatalf("approval not linked: %+v", requests)
	}
	orderID := requests[0].Order.ID
	ctx := context.Background()
	plan, e := c.f.store.CurrentPlan(ctx, input.Date)
	if e != nil {
		t.Fatal(e)
	}
	var visit dto.Stop
	for _, route := range plan.Routes {
		for _, stop := range route.Stops {
			if stop.OrderId != nil && *stop.OrderId == orderID {
				if route.StaffId != "ST-001" {
					t.Fatal("unexpected staff")
				}
				visit = stop
			}
		}
	}
	execute := "/ui/v1/days/2026-10-26/orders/" + orderID + "/execution"
	c.call("technician", "POST", execute, map[string]string{"action": "START"}, 409)
	start, e := time.Parse(time.RFC3339, string(visit.Start))
	if e != nil {
		t.Fatal(e)
	}
	c.call("dispatcher", "POST", "/ui/v1/days/2026-10-26/events", map[string]string{"event_id": "workflow-clock", "type": "TIME_TICK", "timestamp": start.Add(time.Minute).Format(time.RFC3339)}, 200)
	c.call("subscriber", "POST", execute, map[string]string{"action": "START"}, 403)
	c.call("technician", "POST", execute, map[string]string{"action": "COMPLETE", "report": "Восстановлена связь"}, 409)
	c.call("technician", "POST", execute, map[string]string{"action": "START"}, 200)
	before, _ := c.f.store.CurrentPlan(ctx, input.Date)
	beforeJSON, _ := json.Marshal(before)
	for i := 0; i < 2; i++ {
		c.call("technician", "POST", execute, map[string]string{"action": "COMPLETE", "report": "Восстановлена связь"}, 200)
	}
	c.call("technician", "POST", "/ui/v1/days/2026-10-26/events", map[string]string{"event_id": "delay-completed", "type": "VISIT_DELAYED", "staff_id": "ST-001", "order_id": orderID, "timestamp": start.Add(time.Minute).Format(time.RFC3339), "expected_end": start.Add(3 * time.Hour).Format(time.RFC3339)}, 409)
	after, _ := c.f.store.CurrentPlan(ctx, input.Date)
	afterJSON, _ := json.Marshal(after)
	if string(beforeJSON) != string(afterJSON) {
		t.Fatal("execution modified frozen organizer schedule")
	}
	requests = nil
	json.Unmarshal(c.call("subscriber", "GET", "/ui/v1/requests", nil, 200).Body.Bytes(), &requests)
	if requests[0].Order.State != "COMPLETED" || requests[0].Order.Report != "Восстановлена связь" {
		t.Fatal("completion not propagated", requests[0].Order)
	}
	var other []ServiceRequest
	json.Unmarshal(c.call("subscriber-two", "GET", "/ui/v1/requests", nil, 200).Body.Bytes(), &other)
	if len(other) != 0 {
		t.Fatal("foreign request leaked")
	}
}
func TestApprovalRollbackAndRejection(t *testing.T) {
	c := accessSetup(t)
	id := newRequest(c)
	path := "/ui/v1/requests/" + id + "/approve"
	ctx := context.Background()
	_, e := c.f.store.Pool.Exec(ctx, `CREATE FUNCTION reject_approval() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'injected approval failure'; END $$; CREATE TRIGGER reject_approval BEFORE INSERT ON request_orders FOR EACH ROW EXECUTE FUNCTION reject_approval()`)
	if e != nil {
		t.Fatal(e)
	}
	c.call("dispatcher", "POST", path, approvalInput(), 500)
	plan, e := c.f.store.CurrentPlan(ctx, "2026-10-26")
	if e != nil {
		t.Fatal(e)
	}
	if plan.PlanVersion != 1 {
		t.Fatal("plan committed without request link")
	}
	var count int
	var status string
	c.f.store.Pool.QueryRow(ctx, "SELECT count(*) FROM events").Scan(&count)
	c.f.store.Pool.QueryRow(ctx, "SELECT status FROM service_requests WHERE id=$1", id).Scan(&status)
	if count != 0 || status != "NEW" {
		t.Fatal("approval only partially rolled back")
	}
	c.call("dispatcher", "PATCH", "/ui/v1/requests/"+id, map[string]string{"status": "REJECTED", "decision_note": "Адрес вне зоны обслуживания"}, 200)
	c.call("dispatcher", "POST", path, approvalInput(), 409)
}

func TestSubscriberSubmissionRetryAndOwnerScope(t *testing.T) {
	c := accessSetup(t)
	input := map[string]string{"client_key": "test-retry-key", "service": "REPAIR", "address": "Учебный адрес 1", "phone": "+79990000000", "description": "Проверка повторной отправки"}
	ids := make(chan string, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			raw := c.call("subscriber", "POST", "/ui/v1/requests", input, 201)
			var result map[string]string
			json.Unmarshal(raw.Body.Bytes(), &result)
			ids <- result["id"]
		}()
	}
	wg.Wait()
	close(ids)
	first := ""
	for id := range ids {
		if first != "" && first != id {
			t.Fatal("retry created a duplicate")
		}
		first = id
	}
	var count int
	c.f.store.Pool.QueryRow(context.Background(), "SELECT count(*) FROM service_requests").Scan(&count)
	if count != 1 {
		t.Fatal("concurrent duplicate requests", count)
	}
	var second map[string]string
	json.Unmarshal(c.call("subscriber-two", "POST", "/ui/v1/requests", input, 201).Body.Bytes(), &second)
	if second["id"] == first {
		t.Fatal("idempotency key crossed account boundary")
	}
	c.call("dispatcher", "PATCH", "/ui/v1/requests/"+first, map[string]string{"status": "IN_REVIEW", "decision_note": "Проверяется"}, 200)
	var retry map[string]string
	json.Unmarshal(c.call("subscriber", "POST", "/ui/v1/requests", input, 201).Body.Bytes(), &retry)
	if retry["status"] != "IN_REVIEW" {
		t.Fatal("retry returned stale request status")
	}
	input["address"] = "Другой адрес 2"
	c.call("subscriber", "POST", "/ui/v1/requests", input, 409)
}

func TestApprovalNormMustMatchBuildingTechnology(t *testing.T) {
	c := accessSetup(t)
	ctx := context.Background()
	in, e := c.f.store.EventInputs(ctx, "2026-10-26")
	if e != nil {
		t.Fatal(e)
	}
	reference := in.Reference
	work := reference.WorkTypes[1]
	work.Code = "INST_FTTB_MKD"
	reference.WorkTypes = append(reference.WorkTypes, work)
	service := application.Service{Store: c.f.store}
	if e = service.Reference(ctx, reference); e != nil {
		t.Fatal(e)
	}
	if e = service.Day(ctx, in.Day.Date, in.Day); e != nil {
		t.Fatal(e)
	}
	if _, e = service.BuildPlan(ctx, in.Day.Date, 3); e != nil {
		t.Fatal(e)
	}
	var created map[string]string
	json.Unmarshal(c.call("subscriber", "POST", "/ui/v1/requests", map[string]string{"service": "INTERNET", "address": "Учебный адрес PON 1", "phone": "+79990000000"}, 201).Body.Bytes(), &created)
	input := approvalInput()
	input.WorkType = "INST_FTTB_MKD"
	input.Slot = dto.TimeWindow{Start: "2026-10-26T13:00:00+05:00", End: "2026-10-26T15:00:00+05:00"}
	c.call("dispatcher", "POST", "/ui/v1/requests/"+created["id"]+"/approve", input, 422)
	input.WorkType = "INST_PON_MKD"
	c.call("dispatcher", "POST", "/ui/v1/requests/"+created["id"]+"/approve", input, 200)
}

func TestCreateShiftCannotReplaceExistingData(t *testing.T) {
	c := accessSetup(t)
	c.call("subscriber", "POST", "/ui/v1/days", map[string]string{"date": "2026-10-27"}, 403)
	c.call("dispatcher", "POST", "/ui/v1/days", map[string]string{"date": "bad-date"}, 400)
	c.call("dispatcher", "POST", "/ui/v1/days", map[string]string{"date": "2026-10-26"}, 409)
	c.call("dispatcher", "POST", "/ui/v1/days", map[string]string{"date": "2026-10-27"}, 201)
	var day dto.DayInput
	json.Unmarshal(c.call("dispatcher", "POST", "/ui/v1/days", map[string]string{"date": "2026-10-28"}, 201).Body.Bytes(), &day)
	if day.Date != "2026-10-28" || len(day.Orders) != 0 || day.Now != "2026-10-28T08:30:00+05:00" {
		t.Fatal("new shift has unexpected initial state", day)
	}
	c.call("dispatcher", "POST", "/ui/v1/days", map[string]string{"date": "2026-10-27"}, 409)
	existing, e := c.f.store.CurrentPlan(context.Background(), "2026-10-26")
	if e != nil || existing.PlanVersion != 1 || len(existing.Routes) != 2 {
		t.Fatal("existing day was replaced", e)
	}
}
