package dto

// Optional Plan extension; the organizer contract explicitly permits extra
// response fields. It is committed in the same JSON snapshot as the plan.
type PlanQuality struct {
	WeightedVisits int64   `json:"weighted_visits"`
	Visits         int64   `json:"visits"`
	TravelSec      int64   `json:"travel_sec"`
	UtilizationCV  float64 `json:"utilization_cv"`
}
type PlanOptimization struct {
	Engine            string       `json:"engine"`
	Status            string       `json:"status"`
	SolverVersion     string       `json:"solver_version,omitempty"`
	Seed              int64        `json:"seed"`
	Baseline          PlanQuality  `json:"baseline"`
	Selected          PlanQuality  `json:"selected"`
	Candidate         *PlanQuality `json:"candidate,omitempty"`
	CandidateModel    string       `json:"candidate_model,omitempty"`
	ProposedVisits    int64        `json:"proposed_visits,omitempty"`
	CandidatesChecked int64        `json:"candidates_checked"`
	SolverMS          int64        `json:"solver_ms,omitempty"`
	Reason            string       `json:"reason,omitempty"`
}
