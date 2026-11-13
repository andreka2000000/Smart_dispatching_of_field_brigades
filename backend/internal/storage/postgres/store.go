package postgres

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"magistral/backend/internal/dto"
)

//go:embed schema.sql
var schema string

type Store struct{ Pool *pgxpool.Pool }

func Open(ctx context.Context, url string) (*Store, error) {
	p, e := pgxpool.New(ctx, url)
	if e != nil {
		return nil, e
	}
	if e = p.Ping(ctx); e != nil {
		p.Close()
		return nil, e
	}
	s := &Store{p}
	e = s.Update(ctx, func(tx pgx.Tx) error { _, e := tx.Exec(ctx, schema); return e })
	if e != nil {
		p.Close()
		return nil, e
	}
	return s, nil
}
func (s *Store) Update(ctx context.Context, fn func(pgx.Tx) error) error {
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	if _, e = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(506001)"); e != nil {
		return e
	}
	if e = fn(tx); e != nil {
		return e
	}
	return tx.Commit(ctx)
}
func Read[T any](ctx context.Context, tx pgx.Tx, table string) (*T, error) {
	// table is a private application constant, never request input.
	var raw []byte
	e := tx.QueryRow(ctx, "SELECT document FROM "+table+" WHERE id=true").Scan(&raw)
	if errors.Is(e, pgx.ErrNoRows) {
		return nil, nil
	}
	if e != nil {
		return nil, e
	}
	var out T
	e = json.Unmarshal(raw, &out)
	return &out, e
}
func Put(ctx context.Context, tx pgx.Tx, table string, value any) error {
	b, e := json.Marshal(value)
	if e != nil {
		return e
	}
	_, e = tx.Exec(ctx, "INSERT INTO "+table+" (id,version,document) VALUES(true,1,$1) ON CONFLICT(id) DO UPDATE SET version="+table+".version+1,document=EXCLUDED.document", b)
	return e
}
func entity(ctx context.Context, tx pgx.Tx, table, id string, v any) error {
	b, e := json.Marshal(v)
	if e != nil {
		return e
	}
	_, e = tx.Exec(ctx, "INSERT INTO "+table+"(id,document) VALUES($1,$2) ON CONFLICT(id) DO NOTHING", id, b)
	return e
}
func ReplaceReference(ctx context.Context, tx pgx.Tx, r dto.ReferenceData) error {
	if _, e := tx.Exec(ctx, "DELETE FROM days; DELETE FROM staff; DELETE FROM service_centers; DELETE FROM access_nodes; DELETE FROM buildings; DELETE FROM work_types"); e != nil {
		return e
	}
	for _, x := range r.ServiceCenters {
		if e := entity(ctx, tx, "service_centers", x.ServiceCenterId, x); e != nil {
			return e
		}
	}
	for _, x := range r.Staff {
		b, _ := json.Marshal(x)
		if _, e := tx.Exec(ctx, "INSERT INTO staff(id,service_center_id,document) VALUES($1,$2,$3)", x.StaffId, x.ServiceCenterId, b); e != nil {
			return e
		}
	}
	for _, x := range r.AccessNodes {
		if e := entity(ctx, tx, "access_nodes", x.AccessNodeId, x); e != nil {
			return e
		}
		for _, b := range x.Buildings {
			if e := entity(ctx, tx, "buildings", b.BuildingId, b); e != nil {
				return e
			}
		}
	}
	for _, x := range r.WorkTypes {
		if e := entity(ctx, tx, "work_types", string(x.Code), x); e != nil {
			return e
		}
	}
	return Put(ctx, tx, "reference_data", r)
}
func ReplaceDay(ctx context.Context, tx pgx.Tx, d dto.DayInput) error {
	if _, e := tx.Exec(ctx, "DELETE FROM days WHERE date=$1", d.Date); e != nil {
		return e
	}
	b, _ := json.Marshal(d)
	if _, e := tx.Exec(ctx, "INSERT INTO days(date,model_now,document) VALUES($1,$2,$3)", d.Date, string(d.Now), b); e != nil {
		return e
	}
	for _, o := range d.Orders {
		b, _ = json.Marshal(o)
		if _, e := tx.Exec(ctx, "INSERT INTO orders(day,id,building_id,work_type_code,document) VALUES($1,$2,$3,$4,$5)", d.Date, o.OrderId, o.BuildingId, string(o.WorkTypeCode), b); e != nil {
			return e
		}
	}
	return nil
}
