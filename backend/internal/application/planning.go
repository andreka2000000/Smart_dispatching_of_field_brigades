package application

import (
	"context"
	"errors"
	"magistral/backend/internal/constraints"
	"magistral/backend/internal/dto"
	"magistral/backend/internal/planning"
	"magistral/backend/internal/storage/postgres"
	"time"
)

func planError(e error) error {
	if errors.Is(e, postgres.ErrMissing) {
		return &Error{404, "PLAN_NOT_FOUND", "Load day, reference and matrix before planning"}
	}
	if errors.Is(e, postgres.ErrEvents) {
		return &Error{409, "EVENTS_ALREADY_ACCEPTED", e.Error()}
	}
	if errors.Is(e, postgres.ErrStale) {
		return &Error{409, "PLANNING_INPUT_CHANGED", "Inputs or plan changed; request a new calculation"}
	}
	return e
}
func (s *Service) BuildPlan(ctx context.Context, date string, limit int64) (dto.Plan, error) {
	started := time.Now()
	if limit < 1 || limit > 300 {
		return dto.Plan{}, &Error{400, "BAD_REQUEST", "time_limit_sec must be between 1 and 300"}
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(limit)*time.Second)
	defer cancel()
	in, e := s.Store.PlanningInputs(ctx, date)
	if e != nil {
		return dto.Plan{}, planError(e)
	}
	w, e := constraints.New(in.Reference, in.Day, in.Matrix)
	if e != nil {
		return dto.Plan{}, e
	}
	// Reserve time for independent validation, serialization and commit.
	reserve := 250 * time.Millisecond
	if limit >= 5 {
		reserve = time.Second
	}
	search, cancelSearch := context.WithDeadline(ctx, started.Add(time.Duration(limit)*time.Second-reserve))
	defer cancelSearch()
	p, e := planning.Build(search, w)
	if e != nil {
		return dto.Plan{}, e
	}
	if e = w.Validate(p); e != nil {
		return dto.Plan{}, e
	}
	ms := time.Since(started).Milliseconds()
	p.ComputationMs = &ms
	if e = s.Store.Publish(ctx, in, p); e != nil {
		return dto.Plan{}, planError(e)
	}
	return p, nil
}
func (s *Service) Plan(ctx context.Context, date string) (dto.Plan, error) {
	p, e := s.Store.CurrentPlan(ctx, date)
	return p, planError(e)
}
