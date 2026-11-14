package application

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	c "magistral/backend/internal/constraints"
	"magistral/backend/internal/domain"
	"magistral/backend/internal/dto"
	"magistral/backend/internal/insights"
	"magistral/backend/internal/planning"
	"magistral/backend/internal/storage/postgres"
	"reflect"
	"strings"
	"time"
)

func eventConflict(code, message string) error { return &Error{409, code, message} }
func (s *Service) Event(ctx context.Context, date string, ev dto.EventInput) (dto.EventResult, error) {
	return s.event(ctx, date, ev, "", nil)
}
func (s *Service) StaffEvent(ctx context.Context, date string, ev dto.EventInput, staff string) (dto.EventResult, error) {
	if staff == "" || ev.StaffId != staff || (ev.Type != "VISIT_DELAYED" && ev.Type != "ORDER_TRANSFERRED" && ev.Type != "STAFF_UNAVAILABLE") {
		return dto.EventResult{}, &Error{403, "FORBIDDEN", "Можно сообщать только об изменениях своих заданий."}
	}
	return s.event(ctx, date, ev, staff, nil)
}

type eventExtension struct {
	prepare func(postgres.EventInputs, *dto.EventInput) error
	commit  func(pgx.Tx, postgres.EventInputs, dto.EventResult) error
}

func (s *Service) event(ctx context.Context, date string, ev dto.EventInput, staffScope string, extension *eventExtension) (dto.EventResult, error) {
	started := time.Now()
	ctx, cancel := context.WithTimeout(ctx, 9*time.Second)
	defer cancel()
	result := dto.EventResult{}
	if strings.TrimSpace(ev.EventId) == "" {
		return result, &Error{400, "BAD_REQUEST", "event_id must not be empty"}
	}
	e := s.Store.WithDayLock(ctx, date, func() error {
		duplicate, e := s.Store.Duplicate(ctx, date, ev.EventId)
		if e != nil {
			return e
		}
		if duplicate != nil {
			result = *duplicate
			return nil
		}
		in, e := s.Store.EventInputs(ctx, date)
		if e != nil {
			if errors.Is(e, postgres.ErrNoPlan) {
				return eventConflict("PLAN_NOT_BUILT", e.Error())
			}
			return planError(e)
		}
		if extension != nil {
			if e = extension.prepare(in, &ev); e != nil {
				return e
			}
		}
		if ev.Type == "VISIT_DELAYED" || ev.Type == "ORDER_TRANSFERRED" {
			var completed bool
			if e = s.Store.Pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM visit_facts WHERE day=$1 AND generation=$2 AND order_id=$3 AND status='COMPLETED')", date, in.Generation, ev.OrderId).Scan(&completed); e != nil {
				return e
			}
			if completed {
				return &Error{409, "VISIT_COMPLETED", "Выполнение уже подтверждено. Задержка или передача завершённого визита невозможна."}
			}
		}
		clock, e := domain.NewClock(string(in.Day.Now))
		if staffScope != "" {
			ev.Timestamp = in.Day.Now
			if ev.Type != "STAFF_UNAVAILABLE" {
				owned := false
				for _, route := range in.Plan.Routes {
					if route.StaffId == staffScope {
						for _, stop := range route.Stops {
							if stop.OrderId != nil && *stop.OrderId == ev.OrderId {
								owned = true
							}
						}
					}
				}
				if !owned {
					return &Error{403, "FORBIDDEN", "Задание не принадлежит этому исполнителю."}
				}
			}
		}
		if e != nil {
			return e
		}
		if _, e = domain.NewClock(string(ev.Timestamp)); e != nil {
			return invalid("Invalid event timestamp: %s", e)
		}
		next, e := clock.Advance(string(ev.Timestamp))
		if e != nil {
			return eventConflict("EVENT_OUT_OF_ORDER", "timestamp must be valid whole seconds and not precede current model time")
		}
		if next.Now.Format("2006-01-02") != date {
			return invalid("Event timestamp belongs to a different model day")
		}
		ev.Timestamp = c.Stamp(next.Now)
		day := in.Day
		day.Orders = append([]dto.Order{}, in.Day.Orders...)
		day.Now = ev.Timestamp
		state := in.State
		state.Init()
		cancelled := map[string]dto.CancelledOrder{}
		for _, x := range in.Plan.Cancelled {
			cancelled[x.OrderId] = x
		}
		pinned := map[string]bool{}
		find := func(id string) (dto.Stop, string, bool) {
			for _, r := range in.Plan.Routes {
				for _, x := range r.Stops {
					if x.Type == "VISIT" && x.OrderId != nil && *x.OrderId == id {
						return x, r.StaffId, true
					}
				}
			}
			return dto.Stop{}, "", false
		}
		orderExists := func(id string) bool {
			for _, o := range day.Orders {
				if o.OrderId == id {
					return true
				}
			}
			return false
		}
		switch ev.Type {
		case "NEW_ORDER":
			if ev.Order == nil {
				return invalid("NEW_ORDER requires order")
			}
			if orderExists(ev.Order.OrderId) {
				return invalid("Order already exists: %s", ev.Order.OrderId)
			}
			candidate := day
			candidate.Orders = []dto.Order{*ev.Order}
			if e = ValidateDay(date, &candidate, in.Reference); e != nil {
				return e
			}
			day.Orders = append(day.Orders, *ev.Order)
			state.NewOrderAt[ev.Order.OrderId] = ev.Timestamp
		case "ORDER_CANCELLED":
			if !orderExists(ev.OrderId) {
				return invalid("Unknown order: %s", ev.OrderId)
			}
			if _, ok := cancelled[ev.OrderId]; ok {
				return invalid("Order already cancelled")
			}
			stop, _, found := find(ev.OrderId)
			if !found || !c.Time(stop.Start).Before(next.Now) {
				cancelled[ev.OrderId] = dto.CancelledOrder{OrderId: ev.OrderId, CancelledAt: ev.Timestamp, EventId: &ev.EventId}
			}
		case "VISIT_DELAYED", "ORDER_TRANSFERRED":
			if state.Transferred[ev.OrderId] {
				return invalid("Visit has already been transferred")
			}
			stop, staff, found := find(ev.OrderId)
			if !found || staff != ev.StaffId || c.Time(stop.Start).After(next.Now) {
				return invalid("Event must refer to an already started visit of this employee")
			}
			var end dto.DateTime
			if ev.Type == "VISIT_DELAYED" {
				if ev.ExpectedEnd == nil {
					return invalid("Missing expected_end")
				}
				end = *ev.ExpectedEnd
				updated, e := domain.NewClock(string(end))
				if e != nil || !updated.Now.After(c.Time(stop.End)) || updated.Now.Before(next.Now) {
					return invalid("expected_end must be later than planned end and not before timestamp")
				}
			} else {
				if ev.ActualEnd == nil {
					return invalid("Missing actual_end")
				}
				end = *ev.ActualEnd
				updated, e := domain.NewClock(string(end))
				if e != nil || updated.Now.Before(c.Time(stop.Start)) || updated.Now.After(next.Now) || updated.Now.After(c.Time(stop.End)) {
					return invalid("actual_end must be between start and min(timestamp, planned end)")
				}
			}
			state.EndOverrides[ev.OrderId] = end
			if ev.Type == "ORDER_TRANSFERRED" {
				state.Transferred[ev.OrderId] = true
			}
			pinned[ev.OrderId] = true
		case "STAFF_UNAVAILABLE":
			found := false
			for _, s := range in.Reference.Staff {
				if s.StaffId == ev.StaffId && !c.Has(day.AbsentStaffIds, s.StaffId) {
					found = true
				}
			}
			if !found {
				return invalid("Unknown or absent employee: %s", ev.StaffId)
			}
			if ev.Until != nil {
				until, e := domain.NewClock(string(*ev.Until))
				if e != nil || !until.Now.After(next.Now) {
					return invalid("until must be later than timestamp")
				}
			}
			state.Unavailable[ev.StaffId] = append(state.Unavailable[ev.StaffId], domain.Unavailability{From: ev.Timestamp, Until: ev.Until})
		case "TIME_TICK":
		default:
			return invalid("Unsupported event type")
		}
		world, e := c.New(in.Reference, day, in.Matrix)
		if e != nil {
			return e
		}
		world.Previous = &in.Plan
		world.EndOverrides = state.EndOverrides
		world.NewOrderAt = state.NewOrderAt
		world.Unavailable = state.Unavailable
		world.Groups = insights.Detect(world, in.Plan.PgpGroups, state.PgpHistory, cancelled, state.Transferred, pinned)
		risks := insights.Forecast(world, in.Plan, cancelled)
		seed := in.Plan
		seed.PgpGroups = world.Groups
		search, cancelSearch := context.WithDeadline(ctx, started.Add(8*time.Second))
		defer cancelSearch()
		plan, e := planning.Repair(search, world, seed, cancelled, pinned)
		if e != nil {
			if search.Err() != nil {
				return eventConflict("EVENT_DEADLINE", "Event calculation exceeded its budget; no changes committed")
			}
			return &Error{422, "EVENT_INFEASIBLE", e.Error()}
		}
		plan.BasedOnEventId = &ev.EventId
		// This snapshot was produced by event repair, not by initial PyVRP search.
		plan.Optimization = nil
		newAlerts := insights.Alerts(world, in.Plan.Alerts, risks, &plan)
		if e = world.Validate(plan); e != nil {
			return &Error{422, "EVENT_INFEASIBLE", e.Error()}
		}
		ms := time.Since(started).Milliseconds()
		plan.ComputationMs = &ms
		changed := []string{}
		for i, r := range plan.Routes {
			if i >= len(in.Plan.Routes) || !reflect.DeepEqual(r, in.Plan.Routes[i]) {
				changed = append(changed, r.StaffId)
			}
		}
		result = dto.EventResult{EventId: ev.EventId, Accepted: true, Duplicate: false, PlanVersion: plan.PlanVersion, AsOf: ev.Timestamp, ProcessingMs: &ms, ChangedStaffIds: changed, NewAlerts: newAlerts}
		hooks := []func(pgx.Tx) error{}
		if extension != nil {
			hooks = append(hooks, func(tx pgx.Tx) error { return extension.commit(tx, in, result) })
		}
		return planError(s.Store.PublishEvent(ctx, in, ev, day, state, plan, result, hooks...))
	})
	if errors.Is(e, context.DeadlineExceeded) || errors.Is(e, context.Canceled) {
		e = eventConflict("EVENT_DEADLINE", "Event cancelled or queue deadline exceeded; retry using the same event_id")
	}
	return result, e
}
func (s *Service) EventJournal(ctx context.Context, date string) ([]json.RawMessage, error) {
	return s.Store.Journal(ctx, date)
}
