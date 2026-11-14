// Read-only comparison: never connects to PostgreSQL or changes a model day.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"magistral/backend/internal/constraints"
	"magistral/backend/internal/dataset"
	"magistral/backend/internal/dto"
	"magistral/backend/internal/insights"
	"magistral/backend/internal/optimization"
	"os"
	"time"
)

func main() {
	if e := run(); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}
func run() error {
	dir := flag.String("data", "/app/data/synthetic", "case input files")
	date := flag.String("date", "2026-10-28", "model date")
	now := flag.String("now", "2026-10-28T08:30:00+05:00", "model time")
	limit := flag.Duration("budget", 30*time.Second, "total search budget including baseline")
	target := flag.String("optimizer", os.Getenv("OPTIMIZER_URL"), "optional optimizer URL")
	output := flag.String("output", "", "optional JSON report file")
	flag.Parse()
	if *limit < time.Second || *limit > 300*time.Second {
		return fmt.Errorf("budget must be 1s..300s")
	}
	data, e := dataset.Load(*dir, *date, *now)
	if e != nil {
		return e
	}
	w, e := constraints.New(data.Reference, data.Day, data.Matrix)
	if e != nil {
		return e
	}
	w.Groups = insights.Detect(w, nil, map[string]dto.PgpGroup{}, nil, nil, nil)
	var client *optimization.Client
	if *target != "" {
		client, e = optimization.New(*target)
		if e != nil {
			return e
		}
	}
	started := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), *limit)
	defer cancel()
	plan, e := optimization.Build(ctx, w, client)
	if e != nil {
		return e
	}
	insights.Alerts(w, nil, nil, &plan)
	if e = w.Validate(plan); e != nil {
		return e
	}
	result := struct {
		Staff, Orders int
		ElapsedMS     int64    `json:"elapsed_ms"`
		Valid         bool     `json:"valid"`
		Plan          dto.Plan `json:"plan"`
	}{len(data.Reference.Staff), len(data.Day.Orders), time.Since(started).Milliseconds(), true, plan}
	raw, e := json.MarshalIndent(result, "", "  ")
	if e != nil {
		return e
	}
	if *output != "" {
		if e = os.WriteFile(*output, raw, 0644); e != nil {
			return e
		}
		fmt.Fprintln(os.Stderr, "Report:", *output)
	}
	if plan.Optimization != nil {
		summary, _ := json.Marshal(plan.Optimization)
		fmt.Println(string(summary))
	} else {
		summary, _ := json.Marshal(plan.Summary)
		fmt.Println(string(summary))
	}
	return nil
}
