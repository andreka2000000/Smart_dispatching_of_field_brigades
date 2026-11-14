package optimization

import (
	"context"
	"fmt"
	c "magistral/backend/internal/constraints"
	"magistral/backend/internal/dto"
	"magistral/backend/internal/planning"
	"time"
)

type outcome struct {
	plan dto.Plan
	err  error
}
type proposals struct {
	plan   *dto.Plan
	report dto.PlanOptimization
}

// Build runs the baseline alongside PyVRP and exact Go reconstruction, sharing
// the caller's wall-clock deadline. Only a strictly better valid plan is chosen.
func Build(ctx context.Context, w *c.World, client *Client) (dto.Plan, error) {
	if client == nil {
		return planning.Build(ctx, w)
	}
	if w.Previous != nil {
		return dto.Plan{}, fmt.Errorf("initial optimization cannot replan event history")
	}
	deadline, ok := ctx.Deadline()
	if !ok {
		return dto.Plan{}, fmt.Errorf("optimization requires a deadline")
	}
	report := dto.PlanOptimization{Engine: "go_baseline", Status: "short_budget", Reason: "at least 5 seconds of search time required"}
	if time.Until(deadline) < 5*time.Second {
		p, e := planning.Build(ctx, w)
		if e != nil {
			return p, e
		}
		report.Baseline = planning.Quality(w, p)
		report.Selected = report.Baseline
		p.Optimization = &report
		return p, nil
	}
	baseline := make(chan outcome, 1)
	go func() { p, e := planning.Build(ctx, w); baseline <- outcome{p, e} }()
	candidate := make(chan proposals, 1)
	go func() { candidate <- search(ctx, w, client) }()
	original, proposed := <-baseline, <-candidate
	if original.err != nil {
		return dto.Plan{}, original.err
	}
	p := original.plan
	report = proposed.report
	report.Engine = "go_baseline"
	report.Baseline = planning.Quality(w, p)
	report.Selected = report.Baseline
	if proposed.plan != nil {
		q := planning.Quality(w, *proposed.plan)
		report.Candidate = &q
		if planning.Better(q, report.Baseline) {
			p = *proposed.plan
			report.Engine = "pyvrp_go"
			report.Status = "improved"
			report.Selected = q
		} else {
			report.Status = "baseline_retained"
		}
	}
	p.Optimization = &report
	return p, nil
}

func search(ctx context.Context, w *c.World, client *Client) proposals {
	out := proposals{report: dto.PlanOptimization{Status: "unavailable", Seed: 0}}
	deadline, _ := ctx.Deadline()
	budget := min(time.Duration(float64(time.Until(deadline))*0.60), 180*time.Second)
	remote, cancel := context.WithTimeout(ctx, budget+time.Second)
	response, e := client.Propose(remote, w, budget)
	cancel()
	if e != nil {
		out.report.Reason = e.Error()
		return out
	}
	out.report.SolverVersion = response.Version
	out.report.SolverMS = response.ElapsedMS
	out.report.Status = "no_valid_candidate"
	for i, proposal := range response.Candidates {
		seed := map[string][]string{}
		valid := true
		for _, r := range proposal.Routes {
			if _, exists := seed[r.StaffID]; exists {
				valid = false
				break
			}
			seed[r.StaffID] = r.OrderIDs
		}
		if !valid {
			out.report.Reason = "duplicate staff in proposal"
			continue
		}
		remaining := time.Until(deadline)
		if remaining <= 0 {
			break
		}
		exact, cancel := context.WithTimeout(ctx, remaining/time.Duration(len(response.Candidates)-i))
		p, e := planning.FromSequences(exact, w, seed)
		cancel()
		if e != nil {
			out.report.Reason = "proposal rejected by Go validator"
			continue
		}
		if e = w.Validate(p); e != nil {
			out.report.Reason = "exact schedule failed independent validation"
			continue
		}
		out.report.CandidatesChecked++
		if out.plan == nil || planning.Better(planning.Quality(w, p), planning.Quality(w, *out.plan)) {
			out.plan = &p
			out.report.CandidateModel = proposal.Model
			out.report.ProposedVisits = proposal.ProposedVisits
		}
	}
	return out
}
