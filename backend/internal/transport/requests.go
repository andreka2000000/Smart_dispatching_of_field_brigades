package transport

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"magistral/backend/internal/application"
	"magistral/backend/internal/auth"
	"magistral/backend/internal/dto"
	"magistral/backend/internal/storage/postgres"
	"net/http"
	"strings"
	"time"
)

type ServiceRequest struct {
	Order          *RequestOrder `json:"order,omitempty"`
	LinkDay        string        `json:"-"`
	LinkGeneration int64         `json:"-"`
	LinkOrder      string        `json:"-"`
	ID             string        `json:"id"`
	OwnerName      string        `json:"owner_name"`
	Service        string        `json:"service"`
	Address        string        `json:"address"`
	Phone          string        `json:"phone"`
	Description    string        `json:"description"`
	PreferredDate  string        `json:"preferred_date"`
	Status         string        `json:"status"`
	DecisionNote   string        `json:"decision_note"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
}
type RequestOrder struct {
	StaffID      string          `json:"staff_id,omitempty"`
	ID           string          `json:"id"`
	Date         string          `json:"date"`
	State        string          `json:"state"`
	Slot         *dto.TimeWindow `json:"slot,omitempty"`
	StaffName    string          `json:"staff_name,omitempty"`
	PlannedStart *dto.DateTime   `json:"planned_start,omitempty"`
	Reason       string          `json:"reason,omitempty"`
	Report       string          `json:"report,omitempty"`
}

func (a *API) requestRoutes(mux *http.ServeMux, guard func([]string, func(http.ResponseWriter, *http.Request, auth.User)) http.HandlerFunc) {
	mux.HandleFunc("GET /ui/v1/requests", guard([]string{auth.Dispatcher, auth.Subscriber}, func(w http.ResponseWriter, r *http.Request, u auth.User) {
		rows, e := a.Service.Store.Pool.Query(r.Context(), `SELECT r.id,u.name,r.service,r.address,r.phone,r.description,r.preferred_date,r.status,r.decision_note,r.created_at,r.updated_at,coalesce(l.day,''),coalesce(l.generation,0),coalesce(l.order_id,'') FROM service_requests r JOIN app_users u ON u.id=r.user_id LEFT JOIN request_orders l ON l.request_id=r.id WHERE ($1='DISPATCHER' OR r.user_id=$2) ORDER BY r.created_at DESC LIMIT 500`, u.Role, u.ID)
		if e != nil {
			fail(w, e)
			return
		}
		defer rows.Close()
		out := []ServiceRequest{}
		for rows.Next() {
			var x ServiceRequest
			if e = rows.Scan(&x.ID, &x.OwnerName, &x.Service, &x.Address, &x.Phone, &x.Description, &x.PreferredDate, &x.Status, &x.DecisionNote, &x.CreatedAt, &x.UpdatedAt, &x.LinkDay, &x.LinkGeneration, &x.LinkOrder); e != nil {
				fail(w, e)
				return
			}
			out = append(out, x)
		}
		if e = rows.Err(); e != nil {
			fail(w, e)
			return
		}
		rows.Close()
		cache := map[string]postgres.Workspace{}
		for i := range out {
			item := &out[i]
			if item.LinkOrder == "" {
				continue
			}
			item.Order = &RequestOrder{ID: item.LinkOrder, Date: item.LinkDay, State: "REQUIRES_REVIEW"}
			data, ok := cache[item.LinkDay]
			if !ok {
				var e error
				data, e = a.Service.Store.Workspace(r.Context(), item.LinkDay)
				if e != nil {
					continue
				}
				cache[item.LinkDay] = data
			}
			if data.Generation != item.LinkGeneration {
				continue
			}
			item.Order.State = "UNASSIGNED"
			for _, o := range data.Day.Orders {
				if o.OrderId == item.LinkOrder {
					slot := o.Slot
					item.Order.Slot = &slot
				}
			}
			if data.Plan != nil {
				for _, route := range data.Plan.Routes {
					for _, stop := range route.Stops {
						if stop.OrderId != nil && *stop.OrderId == item.LinkOrder {
							item.Order.State = "SCHEDULED"
							if u.Role == auth.Dispatcher {
								item.Order.StaffID = route.StaffId
							}
							start := stop.Start
							item.Order.PlannedStart = &start
							for _, s := range data.Reference.Staff {
								if s.StaffId == route.StaffId {
									if s.DisplayName != nil {
										item.Order.StaffName = *s.DisplayName
									}
									if item.Order.StaffName == "" {
										item.Order.StaffName = s.StaffId
									}
								}
							}
						}
					}
				}
				for _, x := range data.Plan.Unassigned {
					if x.OrderId == item.LinkOrder {
						item.Order.Reason = x.ReasonText
					}
				}
				for _, g := range data.Plan.PgpGroups {
					for _, id := range g.OrderIds {
						if id == item.LinkOrder {
							item.Order.State = "STATION"
						}
					}
				}
				for _, x := range data.Plan.Cancelled {
					if x.OrderId == item.LinkOrder {
						item.Order.State = "CANCELLED"
					}
				}
			}
			for _, f := range data.Facts {
				if f.OrderID == item.LinkOrder {
					item.Order.State = f.Status
					item.Order.Report = f.Report
				}
			}
		}
		write(w, 200, out)
	}))
	mux.HandleFunc("POST /ui/v1/requests", guard([]string{auth.Subscriber}, func(w http.ResponseWriter, r *http.Request, u auth.User) {
		var in struct {
			ClientKey     string `json:"client_key"`
			Service       string `json:"service"`
			Address       string `json:"address"`
			Phone         string `json:"phone"`
			Description   string `json:"description"`
			PreferredDate string `json:"preferred_date"`
		}
		if !readBody(w, r, &in) {
			return
		}
		in.Address = strings.TrimSpace(in.Address)
		in.Phone = strings.TrimSpace(in.Phone)
		in.Description = strings.TrimSpace(in.Description)
		in.ClientKey = strings.TrimSpace(in.ClientKey)
		if len(in.ClientKey) > 100 {
			fail(w, &application.Error{Status: 400, Code: "INVALID_REQUEST", Message: "Слишком длинный идентификатор отправки."})
			return
		}
		_, dateErr := time.Parse("2006-01-02", in.PreferredDate)
		if (in.Service != "INTERNET" && in.Service != "REPAIR" && in.Service != "TV") || len([]rune(in.Address)) < 5 || len([]rune(in.Address)) > 300 || len(in.Phone) < 7 || len(in.Phone) > 40 || len([]rune(in.Description)) > 2000 || (in.PreferredDate != "" && dateErr != nil) {
			fail(w, &application.Error{Status: 400, Code: "INVALID_REQUEST", Message: "Проверьте услугу, адрес, телефон и желаемую дату."})
			return
		}
		id := "REQ-" + auth.Random()[:16]
		key := in.ClientKey
		if key == "" {
			key = auth.Random()
		}
		in.ClientKey = ""
		canonical, _ := json.Marshal(in)
		digest := sha256.Sum256(canonical)
		payloadHash := fmt.Sprintf("%x", digest)
		var requestStatus string
		e := a.Service.Store.Pool.QueryRow(r.Context(), "INSERT INTO service_requests(id,user_id,service,address,phone,description,preferred_date,client_key,payload_hash) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9) ON CONFLICT(user_id,client_key) WHERE client_key IS NOT NULL DO NOTHING RETURNING id,status", id, u.ID, in.Service, in.Address, in.Phone, in.Description, in.PreferredDate, key, payloadHash).Scan(&id, &requestStatus)
		if errors.Is(e, pgx.ErrNoRows) {
			var storedHash string
			e = a.Service.Store.Pool.QueryRow(r.Context(), "SELECT id,payload_hash,status FROM service_requests WHERE user_id=$1 AND client_key=$2", u.ID, key).Scan(&id, &storedHash, &requestStatus)
			if e == nil && storedHash != payloadHash {
				fail(w, &application.Error{Status: 409, Code: "IDEMPOTENCY_CONFLICT", Message: "Этот идентификатор уже использован для другого обращения."})
				return
			}
		}
		if e != nil {
			fail(w, e)
			return
		}
		write(w, 201, map[string]string{"id": id, "status": requestStatus})
	}))
	mux.HandleFunc("PATCH /ui/v1/requests/{id}", guard([]string{auth.Dispatcher}, func(w http.ResponseWriter, r *http.Request, _ auth.User) {
		var in struct {
			Status string `json:"status"`
			Note   string `json:"decision_note"`
		}
		if !readBody(w, r, &in) {
			return
		}
		in.Note = strings.TrimSpace(in.Note)
		if (in.Status != "IN_REVIEW" && in.Status != "CLOSED" && in.Status != "REJECTED") || len([]rune(in.Note)) > 2000 || ((in.Status == "CLOSED" || in.Status == "REJECTED") && len([]rune(in.Note)) < 3) {
			fail(w, &application.Error{Status: 400, Code: "INVALID_REQUEST_STATUS", Message: "Для закрытия обращения добавьте комментарий."})
			return
		}
		result, e := a.Service.Store.Pool.Exec(r.Context(), "UPDATE service_requests SET status=$2,decision_note=$3,updated_at=now() WHERE id=$1 AND status IN ('NEW','IN_REVIEW')", r.PathValue("id"), in.Status, in.Note)
		if e != nil {
			fail(w, e)
			return
		}
		if result.RowsAffected() == 0 {
			fail(w, &application.Error{Status: 409, Code: "REQUEST_CHANGED", Message: "Обращение уже закрыто или не найдено. Обновите список."})
			return
		}
		write(w, 200, map[string]bool{"saved": true})
	}))
	mux.HandleFunc("POST /ui/v1/requests/{id}/approve", guard([]string{auth.Dispatcher}, func(w http.ResponseWriter, r *http.Request, _ auth.User) {
		var in application.Approval
		if !readBody(w, r, &in) {
			return
		}
		result, e := a.Service.Approve(r.Context(), r.PathValue("id"), in)
		if e != nil {
			fail(w, e)
			return
		}
		write(w, 200, result)
	}))
}
