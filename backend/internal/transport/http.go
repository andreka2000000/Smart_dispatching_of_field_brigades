package transport

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	"io"
	"log/slog"
	"magistral/backend/internal/application"
	"magistral/backend/internal/contract"
	"magistral/backend/internal/dto"
	"net/http"
	"time"
	"unicode/utf8"
)

type API struct {
	Service   *application.Service
	Validator *contract.Validator
}

func write(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
func fail(w http.ResponseWriter, e error) {
	var a *application.Error
	if !errors.As(e, &a) {
		slog.Error("request failed", "error", e)
		a = &application.Error{Status: 500, Code: "INTERNAL_ERROR", Message: "Internal server error"}
	}
	write(w, a.Status, map[string]any{"error": map[string]any{"code": a.Code, "message": a.Message}})
}
func (a *API) decode(w http.ResponseWriter, r *http.Request, name string, v any) bool {
	raw, e := io.ReadAll(http.MaxBytesReader(w, r.Body, 100*1024*1024))
	if e == nil && !utf8.Valid(raw) {
		e = errors.New("JSON must be UTF-8")
	}
	if e == nil {
		e = a.Validator.Decode(raw, name, v)
	}
	if e != nil {
		fail(w, &application.Error{Status: 400, Code: "BAD_REQUEST", Message: e.Error()})
		return false
	}
	return true
}
func loaded(w http.ResponseWriter, c map[string]int) {
	write(w, 200, map[string]any{"status": "loaded", "counts": c})
}
func (a *API) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/health", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if e := a.Service.Store.Pool.Ping(ctx); e != nil {
			fail(w, &application.Error{Status: 503, Code: "NOT_READY", Message: "Database unavailable"})
			return
		}
		write(w, 200, map[string]string{"status": "ok", "team": "Магистраль", "version": "0.2.0"})
	})
	mux.HandleFunc("PUT /api/v1/reference", func(w http.ResponseWriter, r *http.Request) {
		var v dto.ReferenceData
		if !a.decode(w, r, "ReferenceData", &v) {
			return
		}
		if e := a.Service.Reference(r.Context(), v); e != nil {
			fail(w, e)
			return
		}
		loaded(w, application.Counts(v))
	})
	mux.HandleFunc("PUT /api/v1/reference/travel-matrix", func(w http.ResponseWriter, r *http.Request) {
		var v dto.TravelMatrix
		if !a.decode(w, r, "TravelMatrix", &v) {
			return
		}
		if e := a.Service.Matrix(r.Context(), v); e != nil {
			fail(w, e)
			return
		}
		loaded(w, map[string]int{"points": len(v.Points)})
	})
	mux.HandleFunc("PUT /api/v1/days/{date}", func(w http.ResponseWriter, r *http.Request) {
		var v dto.DayInput
		if !a.decode(w, r, "DayInput", &v) {
			return
		}
		if _, e := time.Parse("2006-01-02", r.PathValue("date")); e != nil {
			fail(w, &application.Error{Status: 400, Code: "BAD_REQUEST", Message: "Invalid date"})
			return
		}
		if e := a.Service.Day(r.Context(), r.PathValue("date"), v); e != nil {
			fail(w, e)
			return
		}
		loaded(w, map[string]int{"orders": len(v.Orders)})
	})
	mux.HandleFunc("GET /api/v1/days/{date}/plan", func(w http.ResponseWriter, r *http.Request) {
		p, e := a.Service.Plan(r.Context(), r.PathValue("date"))
		if e != nil {
			fail(w, e)
			return
		}
		write(w, 200, p)
	})
	mux.HandleFunc("POST /api/v1/days/{date}/plan", func(w http.ResponseWriter, r *http.Request) {
		limit := int64(300)
		raw, e := io.ReadAll(http.MaxBytesReader(w, r.Body, 100*1024*1024))
		if e != nil {
			fail(w, &application.Error{Status: 400, Code: "BAD_REQUEST", Message: e.Error()})
			return
		}
		if len(raw) > 0 {
			var request dto.PlanRequest
			if !utf8.Valid(raw) {
				fail(w, &application.Error{Status: 400, Code: "BAD_REQUEST", Message: "JSON must be UTF-8"})
				return
			}
			if e = a.Validator.Decode(raw, "PlanRequest", &request); e != nil {
				fail(w, &application.Error{Status: 400, Code: "BAD_REQUEST", Message: e.Error()})
				return
			}
			if request.TimeLimitSec != nil {
				limit = *request.TimeLimitSec
			}
		}
		p, e := a.Service.BuildPlan(r.Context(), r.PathValue("date"), limit)
		if e != nil {
			fail(w, e)
			return
		}
		write(w, 200, p)
	})
	mux.HandleFunc("GET /ui/v1/reference", func(w http.ResponseWriter, r *http.Request) {
		v, e := a.Service.ReadReference(r.Context())
		if errors.Is(e, pgx.ErrNoRows) {
			fail(w, &application.Error{Status: 404, Code: "REFERENCE_NOT_FOUND", Message: "Reference not loaded"})
			return
		}
		if e != nil {
			fail(w, e)
			return
		}
		write(w, 200, v)
	})
	mux.HandleFunc("GET /ui/v1/days/{date}", func(w http.ResponseWriter, r *http.Request) {
		var raw []byte
		var generation int64
		var now time.Time
		e := a.Service.Store.Pool.QueryRow(r.Context(), "SELECT document,generation,model_now FROM days WHERE date::text=$1", r.PathValue("date")).Scan(&raw, &generation, &now)
		if errors.Is(e, pgx.ErrNoRows) {
			fail(w, &application.Error{Status: 404, Code: "DAY_NOT_FOUND", Message: "Day not loaded"})
			return
		}
		if e != nil {
			fail(w, e)
			return
		}
		write(w, 200, map[string]any{"data": json.RawMessage(raw), "generation": generation, "as_of": now.In(time.FixedZone("UTC+5", 18000)).Format(time.RFC3339)})
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fail(w, &application.Error{Status: 404, Code: "NOT_FOUND", Message: "Endpoint not found"})
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if e := recover(); e != nil {
				slog.Error("panic", "error", e)
				fail(w, errors.New("unexpected panic"))
			}
		}()
		mux.ServeHTTP(w, r)
	})
}
