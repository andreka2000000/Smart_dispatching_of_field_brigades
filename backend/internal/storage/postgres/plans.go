package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	"magistral/backend/internal/dto"
)

var ErrMissing = errors.New("day or reference not loaded")
var ErrStale = errors.New("planning inputs changed during calculation")
var ErrEvents = errors.New("initial plan cannot be rebuilt after accepted events")

type Inputs struct {
	Day                                                               dto.DayInput
	Reference                                                         dto.ReferenceData
	Matrix                                                            dto.TravelMatrix
	Generation, Revision, ReferenceVersion, MatrixVersion, EventCount int64
}

func (s *Store) PlanningInputs(ctx context.Context, date string) (Inputs, error) {
	in := Inputs{}
	tx, e := s.Pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if e != nil {
		return in, e
	}
	defer tx.Rollback(ctx)
	var day, ref, matrix []byte
	e = tx.QueryRow(ctx, `SELECT d.document,d.generation,d.plan_revision,d.event_count,r.document,r.version,m.document,m.version FROM days d CROSS JOIN reference_data r CROSS JOIN travel_matrices m WHERE d.date::text=$1`, date).Scan(&day, &in.Generation, &in.Revision, &in.EventCount, &ref, &in.ReferenceVersion, &matrix, &in.MatrixVersion)
	if errors.Is(e, pgx.ErrNoRows) {
		return in, ErrMissing
	}
	if e != nil {
		return in, e
	}
	if in.EventCount > 0 {
		return in, ErrEvents
	}
	if e = json.Unmarshal(day, &in.Day); e != nil {
		return in, e
	}
	if e = json.Unmarshal(ref, &in.Reference); e != nil {
		return in, e
	}
	if e = json.Unmarshal(matrix, &in.Matrix); e != nil {
		return in, e
	}
	return in, tx.Commit(ctx)
}
func (s *Store) Publish(ctx context.Context, in Inputs, p dto.Plan) error {
	return s.Update(ctx, func(tx pgx.Tx) error {
		var gen, revision, events, rv, mv int64
		e := tx.QueryRow(ctx, `SELECT d.generation,d.plan_revision,d.event_count,r.version,m.version FROM days d CROSS JOIN reference_data r CROSS JOIN travel_matrices m WHERE d.date::text=$1`, in.Day.Date).Scan(&gen, &revision, &events, &rv, &mv)
		if errors.Is(e, pgx.ErrNoRows) {
			return ErrStale
		}
		if e != nil {
			return e
		}
		if events > 0 {
			return ErrEvents
		}
		if gen != in.Generation || revision != in.Revision || rv != in.ReferenceVersion || mv != in.MatrixVersion {
			return ErrStale
		}
		b, e := json.Marshal(p)
		if e != nil {
			return e
		}
		if _, e = tx.Exec(ctx, `INSERT INTO plan_snapshots(day,generation,revision,plan_version,reference_version,matrix_version,document) VALUES($1,$2,$3,$4,$5,$6,$7)`, in.Day.Date, gen, revision+1, p.PlanVersion, rv, mv, b); e != nil {
			return e
		}
		_, e = tx.Exec(ctx, `UPDATE days SET plan_revision=plan_revision+1,current_version=$2 WHERE date::text=$1`, in.Day.Date, p.PlanVersion)
		return e
	})
}
func (s *Store) CurrentPlan(ctx context.Context, date string) (dto.Plan, error) {
	var raw []byte
	var p dto.Plan
	e := s.Pool.QueryRow(ctx, `SELECT p.document FROM days d JOIN plan_snapshots p ON p.day=d.date AND p.generation=d.generation AND p.revision=d.plan_revision WHERE d.date::text=$1 AND d.current_version>0`, date).Scan(&raw)
	if errors.Is(e, pgx.ErrNoRows) {
		return p, ErrMissing
	}
	if e != nil {
		return p, e
	}
	e = json.Unmarshal(raw, &p)
	return p, e
}
