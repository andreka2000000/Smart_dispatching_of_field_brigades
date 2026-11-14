package transport

import (
	"encoding/json"
	"magistral/backend/internal/storage/postgres"
	"testing"
)

func TestWorkspaceSnapshot(t *testing.T) {
	f := eventFixture(t)
	var days []postgres.DaySummary
	json.Unmarshal(f.call("GET", "/ui/v1/days", nil, 200), &days)
	if len(days) != 1 || days[0].Date != "2026-10-26" {
		t.Fatal("loaded days missing")
	}
	var w postgres.Workspace
	json.Unmarshal(f.call("GET", "/ui/v1/days/2026-10-26/workspace", nil, 200), &w)
	if w.Plan != nil || len(w.Day.Orders) != 2 || len(w.Reference.Staff) != 2 || w.Events == nil {
		t.Fatal("workspace before build invalid")
	}
	f.call("GET", "/ui/v1/days/2099-01-01/workspace", nil, 404)
	f.build()
	f.event(tick("workspace-test", "08:45:00"), 200)
	json.Unmarshal(f.call("GET", "/ui/v1/days/2026-10-26/workspace", nil, 200), &w)
	if w.Plan == nil || w.Plan.PlanVersion != 2 || w.Plan.AsOf != w.Day.Now || w.EventCount != 1 || len(w.Events) != 1 || w.Events[0].EventId != "workspace-test" || w.Generation == 0 {
		t.Fatal("workspace is not coherent")
	}
}
