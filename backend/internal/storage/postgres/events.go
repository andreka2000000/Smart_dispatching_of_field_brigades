package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	"magistral/backend/internal/domain"
	"magistral/backend/internal/dto"
	"time"
)

var ErrNoPlan = errors.New("build an initial plan before accepting events")

// A session lock serializes events across API processes without holding a SQL
// transaction open during solving. Cancellation also covers waiting in queue.
func (s *Store) WithDayLock(ctx context.Context, date string, fn func() error) error {
	key := "magistral:events:" + date
	var held *pgx.Conn
	for {
		conn, e := s.Pool.Acquire(ctx)
		if e != nil {
			return e
		}
		var locked bool
		if e := conn.QueryRow(ctx, "SELECT pg_try_advisory_lock(hashtextextended($1,0))", key).Scan(&locked); e != nil {
			conn.Release()
			return e
		}
		if locked {
			// The lock holder must not consume a pool slot while the solver
			// acquires other connections. Waiters release theirs immediately.
			held = conn.Hijack()
			break
		}
		conn.Release()
		timer := time.NewTimer(20 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		// Closing the dedicated session releases its lock even on failures.
		held.Close(cleanup)
	}()
	return fn()
}

type EventInputs struct {
	Inputs
	Plan  dto.Plan
	State domain.EventState
}

func (s *Store) Duplicate(ctx context.Context, date, id string) (*dto.EventResult, error) {
	var raw []byte
	e := s.Pool.QueryRow(ctx, `SELECT e.result FROM events e JOIN days d ON d.date=e.day AND d.generation=e.generation WHERE d.date::text=$1 AND e.event_id=$2`, date, id).Scan(&raw)
	if errors.Is(e, pgx.ErrNoRows) {
		return nil, nil
	}
	if e != nil {
		return nil, e
	}
	var r dto.EventResult
	e = json.Unmarshal(raw, &r)
	r.Duplicate = true
	return &r, e
}
func (s *Store) EventInputs(ctx context.Context, date string) (EventInputs, error) {
	in := EventInputs{}
	tx, e := s.Pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if e != nil {
		return in, e
	}
	defer tx.Rollback(ctx)
	var day, ref, matrix, state, plan []byte
	e = tx.QueryRow(ctx, `SELECT d.document,d.generation,d.plan_revision,d.event_count,d.event_state,r.document,r.version,m.document,m.version FROM days d CROSS JOIN reference_data r CROSS JOIN travel_matrices m WHERE d.date::text=$1`, date).Scan(&day, &in.Generation, &in.Revision, &in.EventCount, &state, &ref, &in.ReferenceVersion, &matrix, &in.MatrixVersion)
	if errors.Is(e, pgx.ErrNoRows) {
		return in, ErrMissing
	}
	if e != nil {
		return in, e
	}
	e = tx.QueryRow(ctx, `SELECT document FROM plan_snapshots WHERE day::text=$1 AND generation=$2 AND revision=$3`, date, in.Generation, in.Revision).Scan(&plan)
	if errors.Is(e, pgx.ErrNoRows) {
		return in, ErrNoPlan
	}
	if e != nil {
		return in, e
	}
	for _, pair := range []struct {
		raw []byte
		out any
	}{{day, &in.Day}, {ref, &in.Reference}, {matrix, &in.Matrix}, {state, &in.State}, {plan, &in.Plan}} {
		if e = json.Unmarshal(pair.raw, pair.out); e != nil {
			return in, e
		}
	}
	in.State.Init()
	return in, tx.Commit(ctx)
}
func (s *Store) PublishEvent(ctx context.Context, in EventInputs, ev dto.EventInput, day dto.DayInput, state domain.EventState, p dto.Plan, result dto.EventResult, hooks ...func(pgx.Tx) error) error {
	return s.Update(ctx, func(tx pgx.Tx) error {
		var gen, rev, rv, mv int64
		e := tx.QueryRow(ctx, `SELECT d.generation,d.plan_revision,r.version,m.version FROM days d CROSS JOIN reference_data r CROSS JOIN travel_matrices m WHERE d.date::text=$1`, day.Date).Scan(&gen, &rev, &rv, &mv)
		if errors.Is(e, pgx.ErrNoRows) {
			return ErrStale
		}
		if e != nil {
			return e
		}
		if gen != in.Generation || rev != in.Revision || rv != in.ReferenceVersion || mv != in.MatrixVersion {
			return ErrStale
		}
		planRaw, e := json.Marshal(p)
		if e != nil {
			return e
		}
		eventRaw, e := json.Marshal(ev)
		if e != nil {
			return e
		}
		resultRaw, e := json.Marshal(result)
		if e != nil {
			return e
		}
		dayRaw, e := json.Marshal(day)
		if e != nil {
			return e
		}
		stateRaw, e := json.Marshal(state)
		if e != nil {
			return e
		}
		if ev.Type == "NEW_ORDER" {
			o := ev.Order
			raw, e := json.Marshal(o)
			if e != nil {
				return e
			}
			if _, e = tx.Exec(ctx, `INSERT INTO orders(day,id,building_id,work_type_code,document) VALUES($1,$2,$3,$4,$5)`, day.Date, o.OrderId, o.BuildingId, string(o.WorkTypeCode), raw); e != nil {
				return e
			}
		}
		if _, e = tx.Exec(ctx, `INSERT INTO plan_snapshots(day,generation,revision,plan_version,reference_version,matrix_version,document) VALUES($1,$2,$3,$4,$5,$6,$7)`, day.Date, gen, rev+1, p.PlanVersion, rv, mv, planRaw); e != nil {
			return e
		}
		if _, e = tx.Exec(ctx, `INSERT INTO events(day,generation,event_id,timestamp,type,plan_version,document,result) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, day.Date, gen, ev.EventId, string(ev.Timestamp), string(ev.Type), p.PlanVersion, eventRaw, resultRaw); e != nil {
			return e
		}
		_, e = tx.Exec(ctx, `UPDATE days SET document=$2,event_state=$3,model_now=$4,plan_revision=plan_revision+1,current_version=$5,event_count=event_count+1 WHERE date::text=$1`, day.Date, dayRaw, stateRaw, string(day.Now), p.PlanVersion)
		if e != nil {
			return e
		}
		for _, hook := range hooks {
			if e = hook(tx); e != nil {
				return e
			}
		}
		return nil
	})
}

// The journal is ordered by committed version, including equal timestamps.
func (s *Store) Journal(ctx context.Context, date string) ([]json.RawMessage, error) {
	rows, e := s.Pool.Query(ctx, `SELECT e.document FROM events e JOIN days d ON d.date=e.day AND d.generation=e.generation WHERE d.date::text=$1 ORDER BY e.plan_version`, date)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []json.RawMessage{}
	for rows.Next() {
		var b []byte
		if e = rows.Scan(&b); e != nil {
			return nil, e
		}
		out = append(out, json.RawMessage(b))
	}
	return out, rows.Err()
}
