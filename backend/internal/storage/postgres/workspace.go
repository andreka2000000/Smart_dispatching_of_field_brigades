package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	"magistral/backend/internal/dto"
)

type DaySummary struct {
	Date       string `json:"date"`
	AsOf       string `json:"as_of"`
	Version    int64  `json:"plan_version"`
	EventCount int64  `json:"event_count"`
}

func (s *Store) Days(ctx context.Context) ([]DaySummary, error) {
	rows, e := s.Pool.Query(ctx, `SELECT date::text,to_char(model_now AT TIME ZONE 'Asia/Yekaterinburg','YYYY-MM-DD"T"HH24:MI:SS')||'+05:00',current_version,event_count FROM days ORDER BY date DESC`)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []DaySummary{}
	for rows.Next() {
		var d DaySummary
		if e = rows.Scan(&d.Date, &d.AsOf, &d.Version, &d.EventCount); e != nil {
			return nil, e
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

type Workspace struct {
	Facts      []VisitFact       `json:"facts"`
	Day        dto.DayInput      `json:"day"`
	Reference  dto.ReferenceData `json:"reference"`
	Plan       *dto.Plan         `json:"plan"`
	Events     []dto.EventInput  `json:"events"`
	Generation int64             `json:"generation"`
	EventCount int64             `json:"event_count"`
}

func (s *Store) Workspace(ctx context.Context, date string) (Workspace, error) {
	out := Workspace{Events: []dto.EventInput{}}
	tx, e := s.Pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if e != nil {
		return out, e
	}
	defer tx.Rollback(ctx)
	var day, ref, plan []byte
	e = tx.QueryRow(ctx, `SELECT d.document,r.document,p.document,d.generation,d.event_count FROM days d CROSS JOIN reference_data r LEFT JOIN plan_snapshots p ON p.day=d.date AND p.generation=d.generation AND p.revision=d.plan_revision AND d.current_version>0 WHERE d.date::text=$1`, date).Scan(&day, &ref, &plan, &out.Generation, &out.EventCount)
	if errors.Is(e, pgx.ErrNoRows) {
		return out, ErrMissing
	}
	if e != nil {
		return out, e
	}
	if e = json.Unmarshal(day, &out.Day); e != nil {
		return out, e
	}
	if e = json.Unmarshal(ref, &out.Reference); e != nil {
		return out, e
	}
	if plan != nil {
		if e = json.Unmarshal(plan, &out.Plan); e != nil {
			return out, e
		}
	}
	rows, e := tx.Query(ctx, `SELECT document FROM events WHERE day::text=$1 AND generation=$2 ORDER BY plan_version`, date, out.Generation)
	if e != nil {
		return out, e
	}
	for rows.Next() {
		var raw []byte
		if e = rows.Scan(&raw); e != nil {
			rows.Close()
			return out, e
		}
		var ev dto.EventInput
		if e = json.Unmarshal(raw, &ev); e != nil {
			rows.Close()
			return out, e
		}
		out.Events = append(out.Events, ev)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return out, e
	}
	rows, e = tx.Query(ctx, "SELECT order_id,staff_id,status,to_char(started_at AT TIME ZONE 'Asia/Yekaterinburg','YYYY-MM-DD\"T\"HH24:MI:SS')||'+05:00',CASE WHEN completed_at IS NULL THEN '' ELSE to_char(completed_at AT TIME ZONE 'Asia/Yekaterinburg','YYYY-MM-DD\"T\"HH24:MI:SS')||'+05:00' END,report FROM visit_facts WHERE day=$1 AND generation=$2", date, out.Generation)
	if e != nil {
		return out, e
	}
	out.Facts = []VisitFact{}
	for rows.Next() {
		var fact VisitFact
		if e = rows.Scan(&fact.OrderID, &fact.StaffID, &fact.Status, &fact.StartedAt, &fact.CompletedAt, &fact.Report); e != nil {
			rows.Close()
			return out, e
		}
		out.Facts = append(out.Facts, fact)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return out, e
	}
	return out, tx.Commit(ctx)
}

type VisitFact struct {
	OrderID     string `json:"order_id"`
	StaffID     string `json:"staff_id"`
	Status      string `json:"status"`
	StartedAt   string `json:"started_at"`
	CompletedAt string `json:"completed_at,omitempty"`
	Report      string `json:"report"`
}
