package transport

import (
	"errors"
	"magistral/backend/internal/application"
	"magistral/backend/internal/dto"
	"magistral/backend/internal/routing"
	"magistral/backend/internal/storage/postgres"
	"net/http"
	"strconv"
)

func (a *API) roadRoute(w http.ResponseWriter, r *http.Request) {
	data, e := a.Service.Store.Workspace(r.Context(), r.PathValue("date"))
	if errors.Is(e, postgres.ErrMissing) {
		fail(w, &application.Error{Status: 404, Code: "DAY_NOT_FOUND", Message: "Day not loaded"})
		return
	}
	if e != nil {
		fail(w, e)
		return
	}
	if data.Plan == nil {
		fail(w, &application.Error{Status: 409, Code: "PLAN_NOT_BUILT", Message: "Build a plan before requesting road geometry"})
		return
	}
	for key, actual := range map[string]int64{"version": data.Plan.PlanVersion, "generation": data.Generation} {
		if raw := r.URL.Query().Get(key); raw != "" {
			expected, err := strconv.ParseInt(raw, 10, 64)
			if err != nil {
				fail(w, &application.Error{Status: 400, Code: "BAD_REQUEST", Message: "Invalid route revision"})
				return
			}
			if expected != actual {
				fail(w, &application.Error{Status: 409, Code: "ROUTE_REVISION_CHANGED", Message: "Refresh the workspace before loading this route"})
				return
			}
		}
	}
	var route *dto.Route
	for i := range data.Plan.Routes {
		if data.Plan.Routes[i].StaffId == r.PathValue("staff") {
			route = &data.Plan.Routes[i]
			break
		}
	}
	if route == nil {
		fail(w, &application.Error{Status: 404, Code: "ROUTE_NOT_FOUND", Message: "Staff route not found"})
		return
	}
	locations := map[string]routing.Point{}
	for _, c := range data.Reference.ServiceCenters {
		locations[c.ServiceCenterId] = routing.Point{c.Lon, c.Lat}
	}
	for _, n := range data.Reference.AccessNodes {
		for _, b := range n.Buildings {
			locations[b.BuildingId] = routing.Point{b.Lon, b.Lat}
		}
	}
	points := []routing.Point{}
	indexes := []int{}
	for i, s := range route.Stops {
		p, ok := locations[s.LocationId]
		if !ok {
			fail(w, &application.Error{Status: 422, Code: "ROUTE_LOCATION_UNKNOWN", Message: "Stop coordinates are missing"})
			return
		}
		if len(points) == 0 || points[len(points)-1] != p {
			points = append(points, p)
			indexes = append(indexes, i)
		}
	}
	path := routing.Path{Geometry: routing.Geometry{Type: "LineString", Coordinates: []routing.Point{}}, Legs: []routing.Leg{}}
	if len(points) > 1 {
		if a.Router == nil {
			fail(w, &application.Error{Status: 503, Code: "ROUTING_UNAVAILABLE", Message: "Local OSRM is not configured"})
			return
		}
		path, e = a.Router.Route(r.Context(), points)
		if e != nil {
			code, status := "ROUTING_UNAVAILABLE", 503
			if errors.Is(e, routing.ErrNoRoute) {
				code, status = "ROAD_ROUTE_NOT_FOUND", 422
			}
			fail(w, &application.Error{Status: status, Code: code, Message: "Road route unavailable; the saved plan is unchanged"})
			return
		}
		path.Legs = append([]routing.Leg(nil), path.Legs...)
		for j := range path.Legs {
			// The source is the final stationary stop before movement (e.g. lunch).
			path.Legs[j].FromSeq = route.Stops[indexes[j+1]-1].Seq
			path.Legs[j].ToSeq = route.Stops[indexes[j+1]].Seq
		}
	}
	w.Header().Set("Cache-Control", "no-store")
	write(w, 200, struct {
		routing.Path
		Staff      string `json:"staff_id"`
		Version    int64  `json:"plan_version"`
		Generation int64  `json:"generation"`
		Provider   string `json:"provider"`
		Traffic    string `json:"traffic"`
	}{path, route.StaffId, data.Plan.PlanVersion, data.Generation, "OSRM", "unavailable"})
}
