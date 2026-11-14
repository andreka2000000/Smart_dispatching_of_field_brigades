package transport

import (
	"encoding/json"
	"errors"
	"io"
	"magistral/backend/internal/application"
	"magistral/backend/internal/auth"
	"magistral/backend/internal/dto"
	"magistral/backend/internal/storage/postgres"
	"net"
	"net/http"
	"strings"
	"time"
)

func forbidden(w http.ResponseWriter) {
	fail(w, &application.Error{Status: 403, Code: "FORBIDDEN", Message: "У вашего аккаунта нет доступа к этому действию."})
}
func token(r *http.Request) string {
	cookie, e := r.Cookie(auth.CookieName)
	if e != nil {
		return ""
	}
	return cookie.Value
}
func readBody(w http.ResponseWriter, r *http.Request, out any) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16384))
	decoder.DisallowUnknownFields()
	if e := decoder.Decode(out); e != nil {
		fail(w, &application.Error{Status: 400, Code: "BAD_REQUEST", Message: "Проверьте поля запроса."})
		return false
	}
	if e := decoder.Decode(new(any)); e != io.EOF {
		fail(w, &application.Error{Status: 400, Code: "BAD_REQUEST", Message: "Ожидается один JSON-объект."})
		return false
	}
	return true
}

// ApplicationHandler is the public, authenticated surface. Handler remains the
// unchanged organizer protocol, bound ONLY to the internal evaluation listener.
func (a *API) ApplicationHandler(manager *auth.Manager, origins []string, secure bool) http.Handler {
	raw := a.Handler()
	mux := http.NewServeMux()
	allowed := map[string]bool{}
	for _, origin := range origins {
		allowed[strings.TrimSpace(origin)] = true
	}
	guard := func(roles []string, next func(http.ResponseWriter, *http.Request, auth.User)) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			user, e := manager.Current(r.Context(), token(r))
			if e != nil {
				fail(w, e)
				return
			}
			for _, role := range roles {
				if user.Role == role {
					next(w, r, user)
					return
				}
			}
			forbidden(w)
		}
	}
	cookie := func(w http.ResponseWriter, value string, maxAge int) {
		http.SetCookie(w, &http.Cookie{Name: auth.CookieName, Value: value, Path: "/", MaxAge: maxAge, HttpOnly: true, Secure: secure, SameSite: http.SameSiteStrictMode})
	}
	mux.HandleFunc("GET /api/v1/health", func(w http.ResponseWriter, r *http.Request) { raw.ServeHTTP(w, r) })
	mux.HandleFunc("POST /auth/v1/session", func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if !readBody(w, r, &in) {
			return
		}
		peer, _, _ := net.SplitHostPort(r.RemoteAddr)
		user, session, e := manager.Login(r.Context(), in.Username, in.Password, peer)
		if e != nil {
			fail(w, e)
			return
		}
		if previous := token(r); previous != "" {
			if e = manager.Logout(r.Context(), previous); e != nil {
				fail(w, e)
				return
			}
		}
		cookie(w, session, int(auth.Lifetime/time.Second))
		write(w, 200, user)
	})
	mux.HandleFunc("GET /auth/v1/session", guard([]string{auth.Dispatcher, auth.Technician, auth.Subscriber}, func(w http.ResponseWriter, r *http.Request, u auth.User) { write(w, 200, u) }))
	mux.HandleFunc("DELETE /auth/v1/session", func(w http.ResponseWriter, r *http.Request) {
		if e := manager.Logout(r.Context(), token(r)); e != nil {
			fail(w, e)
			return
		}
		cookie(w, "", -1)
		write(w, 200, map[string]bool{"logged_out": true})
	})
	dispatcher := []string{auth.Dispatcher}
	mux.HandleFunc("POST /ui/v1/days", guard(dispatcher, func(w http.ResponseWriter, r *http.Request, _ auth.User) {
		var in struct {
			Date string `json:"date"`
		}
		if !readBody(w, r, &in) {
			return
		}
		day, e := a.Service.CreateDay(r.Context(), in.Date)
		if e != nil {
			fail(w, e)
			return
		}
		write(w, 201, day)
	}))
	team := []string{auth.Dispatcher, auth.Technician}
	mux.Handle("/api/v1/", guard(dispatcher, func(w http.ResponseWriter, r *http.Request, _ auth.User) { raw.ServeHTTP(w, r) }))
	mux.HandleFunc("GET /ui/v1/days", guard(team, func(w http.ResponseWriter, r *http.Request, u auth.User) {
		days, e := a.Service.Store.Days(r.Context())
		if e != nil {
			fail(w, e)
			return
		}
		if u.Role == auth.Technician {
			for i := range days {
				days[i].EventCount = 0
			}
		}
		write(w, 200, days)
	}))
	mux.HandleFunc("GET /ui/v1/days/{date}/workspace", guard(team, func(w http.ResponseWriter, r *http.Request, u auth.User) {
		data, e := a.Service.Store.Workspace(r.Context(), r.PathValue("date"))
		if errors.Is(e, postgres.ErrMissing) {
			e = &application.Error{Status: 404, Code: "DAY_NOT_FOUND", Message: "Смена не загружена."}
		}
		if e != nil {
			fail(w, e)
			return
		}
		if u.Role == auth.Technician {
			data = personalWorkspace(data, u.StaffID)
		}
		write(w, 200, data)
	}))
	mux.HandleFunc("GET /ui/v1/days/{date}/staff/{staff}/road-route", guard(team, func(w http.ResponseWriter, r *http.Request, u auth.User) {
		if u.Role == auth.Technician && r.PathValue("staff") != u.StaffID {
			forbidden(w)
			return
		}
		a.roadRoute(w, r)
	}))
	mux.HandleFunc("POST /ui/v1/days/{date}/events", guard(team, func(w http.ResponseWriter, r *http.Request, u auth.User) {
		var ev dto.EventInput
		if !a.decode(w, r, "Event", &ev) {
			return
		}
		if strings.TrimSpace(ev.EventId) == "" || len(ev.EventId) > 200 {
			fail(w, &application.Error{Status: 400, Code: "BAD_REQUEST", Message: "Укажите идентификатор события длиной до 200 байт."})
			return
		}
		var result dto.EventResult
		var e error
		if u.Role == auth.Technician {
			ev.EventId = "technician:" + u.ID + ":" + ev.EventId
			result, e = a.Service.StaffEvent(r.Context(), r.PathValue("date"), ev, u.StaffID)
			result.NewAlerts = []dto.Alert{}
			result.ChangedStaffIds = []string{u.StaffID}
		} else {
			result, e = a.Service.Event(r.Context(), r.PathValue("date"), ev)
		}
		if e != nil {
			fail(w, e)
			return
		}
		write(w, 200, result)
	}))
	mux.HandleFunc("GET /ui/v1/users", guard(dispatcher, func(w http.ResponseWriter, r *http.Request, _ auth.User) {
		users, e := manager.List(r.Context())
		if e != nil {
			fail(w, e)
			return
		}
		write(w, 200, users)
	}))
	mux.HandleFunc("POST /ui/v1/users", guard(dispatcher, func(w http.ResponseWriter, r *http.Request, _ auth.User) {
		var in auth.NewUser
		if !readBody(w, r, &in) {
			return
		}
		user, e := manager.Create(r.Context(), in)
		if e != nil {
			fail(w, e)
			return
		}
		write(w, 201, user)
	}))
	mux.HandleFunc("PATCH /ui/v1/users/{id}", guard(dispatcher, func(w http.ResponseWriter, r *http.Request, u auth.User) {
		var in struct {
			Disabled *bool `json:"disabled"`
		}
		if !readBody(w, r, &in) {
			return
		}
		if in.Disabled == nil {
			fail(w, &application.Error{Status: 400, Code: "BAD_REQUEST", Message: "Укажите состояние доступа."})
			return
		}
		if e := manager.Disable(r.Context(), u.ID, r.PathValue("id"), *in.Disabled); e != nil {
			fail(w, e)
			return
		}
		write(w, 200, map[string]bool{"saved": true})
	}))
	a.requestRoutes(mux, guard)
	mux.HandleFunc("POST /ui/v1/days/{date}/orders/{order}/execution", guard([]string{auth.Technician}, func(w http.ResponseWriter, r *http.Request, u auth.User) {
		var in struct {
			Action string `json:"action"`
			Report string `json:"report"`
		}
		if !readBody(w, r, &in) {
			return
		}
		if e := a.Service.Execution(r.Context(), r.PathValue("date"), r.PathValue("order"), u.StaffID, in.Action, in.Report); e != nil {
			fail(w, e)
			return
		}
		write(w, 200, map[string]bool{"saved": true})
	}))
	// Default deny: adding a raw UI endpoint does not silently grant access.
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { http.NotFound(w, r) })
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		if r.Method != "GET" && r.Method != "HEAD" && r.Method != "OPTIONS" {
			if origin := r.Header.Get("Origin"); (origin != "" && !allowed[origin]) || r.Header.Get("Sec-Fetch-Site") == "cross-site" {
				fail(w, &application.Error{Status: 403, Code: "ORIGIN_REJECTED", Message: "Источник запроса не разрешён."})
				return
			}
		}
		mux.ServeHTTP(w, r)
	})
}

// An explicit DTO projection prevents leaks of other staff, orders, event
// payloads, PGP membership, optimization totals and suggested assignees.
func personalWorkspace(data postgres.Workspace, staffID string) postgres.Workspace {
	personalFacts := []postgres.VisitFact{}
	for _, f := range data.Facts {
		if f.StaffID == staffID {
			personalFacts = append(personalFacts, f)
		}
	}
	data.Facts = personalFacts
	ownOrders, points := map[string]bool{}, map[string]bool{}
	if data.Plan != nil {
		routes := []dto.Route{}
		for _, r := range data.Plan.Routes {
			if r.StaffId == staffID {
				routes = append(routes, r)
				for _, s := range r.Stops {
					points[s.LocationId] = true
					if s.OrderId != nil {
						ownOrders[*s.OrderId] = true
					}
				}
			}
		}
		data.Plan = &dto.Plan{Date: data.Plan.Date, PlanVersion: data.Plan.PlanVersion, AsOf: data.Plan.AsOf, Routes: routes, Unassigned: []dto.UnassignedOrder{}, PgpGroups: []dto.PgpGroup{}, Cancelled: []dto.CancelledOrder{}, Alerts: []dto.Alert{}}
	}
	staff := []dto.Staff{}
	for _, s := range data.Reference.Staff {
		if s.StaffId == staffID {
			staff = append(staff, s)
			points[s.ServiceCenterId] = true
		}
	}
	data.Reference.Staff = staff
	orders := []dto.Order{}
	for _, o := range data.Day.Orders {
		if ownOrders[o.OrderId] {
			orders = append(orders, o)
			points[o.BuildingId] = true
		}
	}
	data.Day.Orders = orders
	data.Day.AbsentStaffIds = []string{}
	nodes := []dto.AccessNode{}
	for _, node := range data.Reference.AccessNodes {
		buildings := []dto.Building{}
		for _, b := range node.Buildings {
			if points[b.BuildingId] {
				buildings = append(buildings, b)
			}
		}
		if len(buildings) > 0 {
			node.Buildings = buildings
			nodes = append(nodes, node)
		}
	}
	data.Reference.AccessNodes = nodes
	centers := []dto.ServiceCenter{}
	for _, center := range data.Reference.ServiceCenters {
		if points[center.ServiceCenterId] {
			centers = append(centers, center)
		}
	}
	data.Reference.ServiceCenters = centers
	data.Events = []dto.EventInput{}
	data.EventCount = 0
	return data
}
