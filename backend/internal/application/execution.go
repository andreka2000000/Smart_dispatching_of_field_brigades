package application

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	c "magistral/backend/internal/constraints"
	"magistral/backend/internal/dto"
	"magistral/backend/internal/storage/postgres"
	"strings"
)

func (s *Service) Execution(ctx context.Context, date, orderID, staffID, action, report string) error {
	if action != "START" && action != "COMPLETE" {
		return invalid("Неизвестное действие.")
	}
	report = strings.TrimSpace(report)
	if len([]rune(report)) > 2000 || (action == "COMPLETE" && len([]rune(report)) < 3) {
		return invalid("Добавьте отчёт о выполнении: 3–2000 символов.")
	}
	return s.Store.WithDayLock(ctx, date, func() error {
		in, e := s.Store.EventInputs(ctx, date)
		if e != nil {
			return planError(e)
		}
		var visit *dto.Stop
		for _, r := range in.Plan.Routes {
			if r.StaffId == staffID {
				for i := range r.Stops {
					if r.Stops[i].Type == "VISIT" && r.Stops[i].OrderId != nil && *r.Stops[i].OrderId == orderID {
						copy := r.Stops[i]
						visit = &copy
					}
				}
			}
		}
		if visit == nil {
			return &Error{403, "FORBIDDEN", "Задание не назначено этому исполнителю."}
		}
		if c.Time(in.Day.Now).Before(c.Time(visit.Start)) {
			return &Error{409, "VISIT_NOT_STARTED", "В модельной смене время визита ещё не наступило."}
		}
		return s.Store.Update(ctx, func(tx pgx.Tx) error {
			var generation, revision int64
			if e := tx.QueryRow(ctx, "SELECT generation,plan_revision FROM days WHERE date::text=$1", date).Scan(&generation, &revision); e != nil {
				return e
			}
			if generation != in.Generation || revision != in.Revision {
				return postgres.ErrStale
			}
			var status string
			e := tx.QueryRow(ctx, "SELECT status FROM visit_facts WHERE day=$1 AND generation=$2 AND order_id=$3", date, generation, orderID).Scan(&status)
			if e != nil && !errors.Is(e, pgx.ErrNoRows) {
				return e
			}
			if action == "START" {
				if e == nil {
					return nil
				}
				_, e = tx.Exec(ctx, "INSERT INTO visit_facts(day,generation,order_id,staff_id,status,started_at) VALUES($1,$2,$3,$4,'IN_PROGRESS',$5)", date, generation, orderID, staffID, string(in.Day.Now))
				return e
			}
			if status == "COMPLETED" {
				return nil
			}
			if status != "IN_PROGRESS" {
				return &Error{409, "VISIT_NOT_STARTED", "Сначала подтвердите начало работы."}
			}
			_, e = tx.Exec(ctx, "UPDATE visit_facts SET status='COMPLETED',completed_at=$4,report=$5 WHERE day=$1 AND generation=$2 AND order_id=$3", date, generation, orderID, string(in.Day.Now), report)
			return e
		})
	})
}
