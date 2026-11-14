package transport

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"magistral/backend/internal/application"
	"magistral/backend/internal/auth"
	"magistral/backend/internal/dto"
	"magistral/backend/internal/storage/postgres"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const testPassword = "Test-only-password-2026!"

type accessCase struct {
	f       *eventsFixture
	manager *auth.Manager
	users   map[string]auth.User
	cookies map[string]*http.Cookie
}

func accessSetup(t *testing.T) *accessCase {
	f := eventFixture(t)
	f.build()
	manager := &auth.Manager{Store: f.store}
	out := &accessCase{f: f, manager: manager, users: map[string]auth.User{}, cookies: map[string]*http.Cookie{}}
	for _, in := range []auth.NewUser{{Username: "dispatcher", Name: "Dispatcher", Role: auth.Dispatcher}, {Username: "technician", Name: "Technician", Role: auth.Technician, StaffID: "ST-001"}, {Username: "subscriber", Name: "Subscriber one", Role: auth.Subscriber}, {Username: "subscriber-two", Name: "Subscriber two", Role: auth.Subscriber}} {
		in.Password = testPassword
		u, e := manager.Create(context.Background(), in)
		if e != nil {
			t.Fatal(e)
		}
		out.users[in.Username] = u
	}
	f.api = (&API{Service: &application.Service{Store: f.store}, Validator: f.validator}).ApplicationHandler(manager, []string{"http://127.0.0.1:3001"}, false)
	for username := range out.users {
		rr := out.call("", http.MethodPost, "/auth/v1/session", map[string]string{"username": username, "password": testPassword}, 200)
		cookies := rr.Result().Cookies()
		if len(cookies) != 1 || !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteStrictMode || cookies[0].MaxAge != 28800 {
			t.Fatal("bad session cookie")
		}
		out.cookies[username] = cookies[0]
	}
	return out
}
func (c *accessCase) call(user, method, path string, body any, status int) *httptest.ResponseRecorder {
	c.f.t.Helper()
	var raw []byte
	if body != nil {
		raw = encode(body)
	}
	r := httptest.NewRequest(method, path, bytes.NewReader(raw))
	if cookie := c.cookies[user]; cookie != nil {
		r.AddCookie(cookie)
	}
	r.Header.Set("Origin", "http://127.0.0.1:3001")
	rr := httptest.NewRecorder()
	c.f.api.ServeHTTP(rr, r)
	if rr.Code != status {
		c.f.t.Fatalf("%s %s as %s: got %d want %d: %s", method, path, user, rr.Code, status, rr.Body.String())
	}
	return rr
}
func TestApplicationRoleIsolationAndSubscriberOwnership(t *testing.T) {
	c := accessSetup(t)
	for _, path := range []string{"/ui/v1/users", dayURL + "/plan", "/ui/v1/days/2026-10-26/workspace", "/ui/v1/requests"} {
		c.call("", "GET", path, nil, 401)
	}
	raw := c.call("technician", "GET", "/ui/v1/days/2026-10-26/workspace", nil, 200).Body.Bytes()
	var own postgres.Workspace
	if e := json.Unmarshal(raw, &own); e != nil {
		t.Fatal(e)
	}
	if len(own.Reference.Staff) != 1 || own.Reference.Staff[0].StaffId != "ST-001" || len(own.Plan.Routes) != 1 || len(own.Day.Orders) != 1 || len(own.Events) != 0 || own.Plan.Summary != nil || strings.Contains(string(raw), "ST-029") || strings.Contains(string(raw), "ORD-20261026-0042") {
		t.Fatal("other staff data leaked", string(raw))
	}
	c.call("technician", "GET", dayURL+"/plan", nil, 403)
	c.call("subscriber", "GET", "/ui/v1/days/2026-10-26/workspace", nil, 403)
	c.call("technician", "GET", "/ui/v1/days/2026-10-26/staff/ST-029/road-route", nil, 403)
	for _, u := range []string{"technician", "subscriber"} {
		c.call(u, "GET", "/ui/v1/users?role=DISPATCHER", nil, 403)
		c.call(u, "POST", dayURL+"/plan", map[string]int{"time_limit_sec": 1}, 403)
	}
	for _, payload := range []map[string]any{
		{"event_id": "forbidden-clock", "type": "TIME_TICK", "timestamp": "2026-10-26T09:30:00+05:00"},
		{"event_id": "other-staff", "type": "STAFF_UNAVAILABLE", "timestamp": "2026-10-26T08:30:00+05:00", "staff_id": "ST-029", "reason": "ILLNESS", "until": nil},
		{"event_id": "other-job", "type": "VISIT_DELAYED", "timestamp": "2026-10-26T13:00:00+05:00", "staff_id": "ST-001", "order_id": "ORD-20261026-0042", "expected_end": "2026-10-26T17:00:00+05:00"},
	} {
		c.call("technician", "POST", "/ui/v1/days/2026-10-26/events", payload, 403)
	}
	stored, e := c.f.store.CurrentPlan(context.Background(), "2026-10-26")
	if e != nil || stored.PlanVersion != 1 {
		t.Fatal("forbidden operation changed plan")
	}
	input := map[string]any{"service": "REPAIR", "address": "Екатеринбург, тестовый дом 1", "phone": "+79990000000", "description": "Контрольная заявка", "preferred_date": "2026-11-20"}
	c.call("subscriber", "POST", "/ui/v1/requests", input, 201)
	input["address"] = "Второй частный адрес"
	c.call("subscriber-two", "POST", "/ui/v1/requests", input, 201)
	var personal []ServiceRequest
	json.Unmarshal(c.call("subscriber", "GET", "/ui/v1/requests", nil, 200).Body.Bytes(), &personal)
	if len(personal) != 1 || personal[0].Address == input["address"] {
		t.Fatal("subscriber ownership leaked")
	}
	c.call("subscriber", "PATCH", "/ui/v1/requests/"+personal[0].ID, map[string]string{"status": "CLOSED", "decision_note": "forbidden"}, 403)
	c.call("technician", "GET", "/ui/v1/requests", nil, 403)
	var all []ServiceRequest
	json.Unmarshal(c.call("dispatcher", "GET", "/ui/v1/requests", nil, 200).Body.Bytes(), &all)
	if len(all) != 2 {
		t.Fatal("dispatcher inbox missing requests")
	}
	c.call("dispatcher", "PATCH", "/ui/v1/requests/"+personal[0].ID, map[string]string{"status": "IN_REVIEW", "decision_note": "Проверяем детали"}, 200)
	c.call("dispatcher", "PATCH", "/ui/v1/requests/"+personal[0].ID, map[string]string{"status": "CLOSED", "decision_note": "Согласован следующий шаг"}, 200)
	body := c.call("subscriber", "GET", "/ui/v1/requests", nil, 200).Body.String()
	if !strings.Contains(body, "Согласован следующий шаг") {
		t.Fatal("reply not visible to owner")
	}
	input["user_id"] = c.users["subscriber-two"].ID
	c.call("subscriber", "POST", "/ui/v1/requests", input, 400)
	listing := c.call("dispatcher", "GET", "/ui/v1/users", nil, 200).Body.String()
	if strings.Contains(listing, "password") || strings.Contains(listing, testPassword) {
		t.Fatal("credential leaked")
	}
}
func TestApplicationSessionRevocationOriginAndLoginLimits(t *testing.T) {
	c := accessSetup(t)
	r := httptest.NewRequest("POST", "/auth/v1/session", strings.NewReader(`{"username":"dispatcher","password":"bad"}`))
	r.Header.Set("Origin", "https://evil.invalid")
	rr := httptest.NewRecorder()
	c.f.api.ServeHTTP(rr, r)
	if rr.Code != 403 {
		t.Fatal("cross-origin login accepted")
	}
	c.call("dispatcher", "PATCH", "/ui/v1/users/"+c.users["dispatcher"].ID, map[string]bool{"disabled": true}, 409)
	c.call("dispatcher", "PATCH", "/ui/v1/users/"+c.users["technician"].ID, map[string]bool{"disabled": true}, 200)
	c.call("technician", "GET", "/auth/v1/session", nil, 401)
	c.call("subscriber", "DELETE", "/auth/v1/session", nil, 200)
	c.call("subscriber", "GET", "/auth/v1/session", nil, 401)
	digest := sha256.Sum256([]byte(c.cookies["subscriber-two"].Value))
	_, e := c.f.store.Pool.Exec(context.Background(), "UPDATE app_sessions SET expires_at=now()-interval '1 second' WHERE token_hash=$1", digest[:])
	if e != nil {
		t.Fatal(e)
	}
	c.call("subscriber-two", "GET", "/auth/v1/session", nil, 401)
	for i := 0; i < 11; i++ {
		expected := 401
		if i == 10 {
			expected = 429
		}
		c.call("", "POST", "/auth/v1/session", map[string]string{"username": "unknown-account", "password": "incorrect"}, expected)
	}
}

func TestTechnicianOwnDelayUsesServerClockAndIsIdempotent(t *testing.T) {
	c := accessSetup(t)
	ctx := context.Background()
	before, e := c.f.store.CurrentPlan(ctx, "2026-10-26")
	if e != nil {
		t.Fatal(e)
	}
	var visit dto.Stop
	for _, route := range before.Routes {
		if route.StaffId == "ST-001" {
			for _, stop := range route.Stops {
				if stop.Type == "VISIT" {
					visit = stop
					break
				}
			}
		}
	}
	start, e := time.Parse(time.RFC3339, string(visit.Start))
	if e != nil {
		t.Fatal(e)
	}
	now := start.Add(time.Minute).Format(time.RFC3339)
	c.call("dispatcher", "POST", "/ui/v1/days/2026-10-26/events", map[string]string{"event_id": "advance-for-owned-visit", "type": "TIME_TICK", "timestamp": now}, 200)
	end, e := time.Parse(time.RFC3339, string(visit.End))
	if e != nil {
		t.Fatal(e)
	}
	input := map[string]string{"event_id": "own-delay", "type": "VISIT_DELAYED", "staff_id": "ST-001", "order_id": *visit.OrderId, "timestamp": "2026-10-26T18:00:00+05:00", "expected_end": end.Add(15 * time.Minute).Format(time.RFC3339)}
	var first dto.EventResult
	json.Unmarshal(c.call("technician", "POST", "/ui/v1/days/2026-10-26/events", input, 200).Body.Bytes(), &first)
	var duplicate dto.EventResult
	json.Unmarshal(c.call("technician", "POST", "/ui/v1/days/2026-10-26/events", input, 200).Body.Bytes(), &duplicate)
	plan, e := c.f.store.CurrentPlan(ctx, "2026-10-26")
	if e != nil {
		t.Fatal(e)
	}
	if !first.Accepted || !duplicate.Duplicate || duplicate.PlanVersion != first.PlanVersion || string(plan.AsOf) != now || len(first.NewAlerts) != 0 {
		t.Fatal("owned event did not preserve scope, clock or deduplication", first, duplicate, plan.AsOf)
	}
}
