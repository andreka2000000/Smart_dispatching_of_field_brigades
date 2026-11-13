// Code generated from contracts/api_spec.yaml; DO NOT EDIT.
package dto

import "encoding/json"

type DateTime string
type LocalTime string
type TimeWindow struct {
	Start DateTime `json:"start"`
	End   DateTime `json:"end"`
}
type Technology string
type Profile string
type WorkTypeCode string
type BuildingType string
type Equipment struct {
	ONT    int64 `json:"ONT"`
	ROUTER int64 `json:"ROUTER"`
	STB    int64 `json:"STB"`
}
type ServiceCenter struct {
	ServiceCenterId string  `json:"service_center_id"`
	Name            string  `json:"name"`
	Address         string  `json:"address"`
	Lat             float64 `json:"lat"`
	Lon             float64 `json:"lon"`
	OpeningHours    struct {
		Open  LocalTime `json:"open"`
		Close LocalTime `json:"close"`
	} `json:"opening_hours"`
	RestockDurationMin int64 `json:"restock_duration_min"`
}
type ServiceCentersFile []ServiceCenter
type Shift struct {
	Start            LocalTime `json:"start"`
	End              LocalTime `json:"end"`
	BreakDurationMin int64     `json:"break_duration_min"`
	BreakWindow      struct {
		From LocalTime `json:"from"`
		To   LocalTime `json:"to"`
	} `json:"break_window"`
}
type Staff struct {
	StaffId           string       `json:"staff_id"`
	DisplayName       *string      `json:"display_name,omitempty"`
	ServiceCenterId   string       `json:"service_center_id"`
	Profiles          []Profile    `json:"profiles"`
	Technologies      []Technology `json:"technologies"`
	HeightPermit      bool         `json:"height_permit"`
	Shift             Shift        `json:"shift"`
	EquipmentStart    Equipment    `json:"equipment_start"`
	EquipmentCapacity Equipment    `json:"equipment_capacity"`
	ExperienceYears   *float64     `json:"experience_years,omitempty"`
}
type StaffFile []Staff
type Building struct {
	BuildingId          string       `json:"building_id"`
	Address             string       `json:"address"`
	Lat                 float64      `json:"lat"`
	Lon                 float64      `json:"lon"`
	BuildingType        BuildingType `json:"building_type"`
	OverheadLine        bool         `json:"overhead_line"`
	ZoneServiceCenterId *string      `json:"zone_service_center_id,omitempty"`
}
type AccessNode struct {
	AccessNodeId string     `json:"access_node_id"`
	NodeType     string     `json:"node_type"`
	Technology   Technology `json:"technology"`
	Name         string     `json:"name"`
	Address      *string    `json:"address,omitempty"`
	Lat          float64    `json:"lat"`
	Lon          float64    `json:"lon"`
	Buildings    []Building `json:"buildings"`
}
type AccessNodesFile []AccessNode
type WorkType struct {
	Code                WorkTypeCode `json:"code"`
	Name                string       `json:"name"`
	Profiles            []Profile    `json:"profiles"`
	DurationMin         int64        `json:"duration_min"`
	MaxStartDelayMin    *int64       `json:"max_start_delay_min"`
	DeadlineKind        string       `json:"deadline_kind"`
	DeadlineValue       *int64       `json:"deadline_value"`
	RepeatDeadlineValue *int64       `json:"repeat_deadline_value"`
	PgpRelevant         bool         `json:"pgp_relevant"`
}
type TravelMatrix struct {
	MatrixId      string    `json:"matrix_id"`
	Points        []string  `json:"points"`
	DurationsSec  [][]int64 `json:"durations_sec"`
	DistancesM    [][]int64 `json:"distances_m"`
	HourlyFactors []struct {
		Hour   int64   `json:"hour"`
		Factor float64 `json:"factor"`
	} `json:"hourly_factors,omitempty"`
}
type ReferenceData struct {
	ServiceCenters []ServiceCenter `json:"service_centers"`
	Staff          []Staff         `json:"staff"`
	AccessNodes    []AccessNode    `json:"access_nodes"`
	WorkTypes      []WorkType      `json:"work_types"`
}
type Order struct {
	OrderId           string       `json:"order_id"`
	WorkTypeCode      WorkTypeCode `json:"work_type_code"`
	Technology        Technology   `json:"technology"`
	BuildingId        string       `json:"building_id"`
	Address           string       `json:"address"`
	Lat               float64      `json:"lat"`
	Lon               float64      `json:"lon"`
	BuildingType      BuildingType `json:"building_type"`
	RequiresHeight    bool         `json:"requires_height"`
	AccessNodeId      string       `json:"access_node_id"`
	RegisteredAt      DateTime     `json:"registered_at"`
	Slot              TimeWindow   `json:"slot"`
	LatestStart       DateTime     `json:"latest_start"`
	Deadline          DateTime     `json:"deadline"`
	RequiredEquipment Equipment    `json:"required_equipment"`
	IsRepeat          bool         `json:"is_repeat"`
	PreviousOrderId   *string      `json:"previous_order_id,omitempty"`
}
type OrderFact struct {
	Outcome           *string   `json:"outcome,omitempty"`
	StaffId           *string   `json:"staff_id,omitempty"`
	ActualStart       *DateTime `json:"actual_start,omitempty"`
	ActualDurationMin *int64    `json:"actual_duration_min,omitempty"`
	PgpIncidentId     *string   `json:"pgp_incident_id,omitempty"`
}
type OrdersFile []Order
type DayInput struct {
	Date           string   `json:"date"`
	Now            DateTime `json:"now"`
	Orders         []Order  `json:"orders"`
	AbsentStaffIds []string `json:"absent_staff_ids,omitempty"`
}
type PlanRequest struct {
	TimeLimitSec *int64 `json:"time_limit_sec,omitempty"`
}
type Plan struct {
	Date           string            `json:"date"`
	PlanVersion    int64             `json:"plan_version"`
	BasedOnEventId *string           `json:"based_on_event_id,omitempty"`
	AsOf           DateTime          `json:"as_of"`
	ComputationMs  *int64            `json:"computation_ms,omitempty"`
	Routes         []Route           `json:"routes"`
	Unassigned     []UnassignedOrder `json:"unassigned"`
	PgpGroups      []PgpGroup        `json:"pgp_groups"`
	Cancelled      []CancelledOrder  `json:"cancelled"`
	Alerts         []Alert           `json:"alerts"`
	Summary        *PlanSummary      `json:"summary,omitempty"`
}
type Route struct {
	StaffId          string       `json:"staff_id"`
	ServiceCenterId  string       `json:"service_center_id"`
	Status           string       `json:"status"`
	UnavailableFrom  *DateTime    `json:"unavailable_from,omitempty"`
	UnavailableUntil *DateTime    `json:"unavailable_until,omitempty"`
	Stops            []Stop       `json:"stops"`
	Totals           *RouteTotals `json:"totals,omitempty"`
}
type Stop struct {
	Seq            int64      `json:"seq"`
	Type           string     `json:"type"`
	LocationId     string     `json:"location_id"`
	OrderId        *string    `json:"order_id,omitempty"`
	Arrival        DateTime   `json:"arrival"`
	Start          DateTime   `json:"start"`
	End            DateTime   `json:"end"`
	TravelSec      *int64     `json:"travel_sec,omitempty"`
	DistanceM      *int64     `json:"distance_m,omitempty"`
	Status         *string    `json:"status,omitempty"`
	EquipmentAfter *Equipment `json:"equipment_after,omitempty"`
	Explanation    *string    `json:"explanation,omitempty"`
}
type RouteTotals struct {
	Visits      *int64   `json:"visits,omitempty"`
	TravelSec   *int64   `json:"travel_sec,omitempty"`
	DistanceM   *int64   `json:"distance_m,omitempty"`
	WorkMin     *int64   `json:"work_min,omitempty"`
	Utilization *float64 `json:"utilization,omitempty"`
}
type UnassignedOrder struct {
	OrderId         string      `json:"order_id"`
	ReasonCode      string      `json:"reason_code"`
	ReasonText      string      `json:"reason_text"`
	SuggestedAction string      `json:"suggested_action"`
	ProposedSlot    *TimeWindow `json:"proposed_slot,omitempty"`
	Since           *DateTime   `json:"since,omitempty"`
}
type PgpGroup struct {
	GroupId      string   `json:"group_id"`
	AccessNodeId string   `json:"access_node_id"`
	OrderIds     []string `json:"order_ids"`
	DetectedAt   DateTime `json:"detected_at"`
	Action       string   `json:"action"`
	Comment      *string  `json:"comment,omitempty"`
}
type CancelledOrder struct {
	OrderId     string   `json:"order_id"`
	CancelledAt DateTime `json:"cancelled_at"`
	EventId     *string  `json:"event_id,omitempty"`
}
type Alert struct {
	AlertId          string      `json:"alert_id"`
	CreatedAt        DateTime    `json:"created_at"`
	Type             string      `json:"type"`
	Status           string      `json:"status"`
	ResolvedAt       *DateTime   `json:"resolved_at,omitempty"`
	OrderId          *string     `json:"order_id,omitempty"`
	GroupId          *string     `json:"group_id,omitempty"`
	StaffId          *string     `json:"staff_id,omitempty"`
	ForecastStart    *DateTime   `json:"forecast_start,omitempty"`
	LatenessMin      *int64      `json:"lateness_min,omitempty"`
	SuggestedAction  string      `json:"suggested_action"`
	SuggestedStaffId *string     `json:"suggested_staff_id,omitempty"`
	ProposedSlot     *TimeWindow `json:"proposed_slot,omitempty"`
	Applied          *bool       `json:"applied,omitempty"`
	Message          string      `json:"message"`
}
type PlanSummary struct {
	OrdersTotal    *int64   `json:"orders_total,omitempty"`
	VisitsPlanned  *int64   `json:"visits_planned,omitempty"`
	Unassigned     *int64   `json:"unassigned,omitempty"`
	PgpOrders      *int64   `json:"pgp_orders,omitempty"`
	Cancelled      *int64   `json:"cancelled,omitempty"`
	TravelSecTotal *int64   `json:"travel_sec_total,omitempty"`
	UtilizationCv  *float64 `json:"utilization_cv,omitempty"`
}
type EventType string
type EventBase struct {
	EventId   string    `json:"event_id"`
	Type      EventType `json:"type"`
	Timestamp DateTime  `json:"timestamp"`
}
type NewOrderEvent json.RawMessage
type OrderCancelledEvent json.RawMessage
type VisitDelayedEvent json.RawMessage
type OrderTransferredEvent json.RawMessage
type StaffUnavailableEvent json.RawMessage
type TimeTickEvent json.RawMessage
type Event json.RawMessage
type EventResult struct {
	EventId         string   `json:"event_id"`
	Accepted        bool     `json:"accepted"`
	Duplicate       bool     `json:"duplicate"`
	PlanVersion     int64    `json:"plan_version"`
	AsOf            DateTime `json:"as_of"`
	ProcessingMs    *int64   `json:"processing_ms,omitempty"`
	ChangedStaffIds []string `json:"changed_staff_ids,omitempty"`
	NewAlerts       []Alert  `json:"new_alerts,omitempty"`
	Message         *string  `json:"message,omitempty"`
}
type EventScenarioFile struct {
	Date     string            `json:"date"`
	Now      DateTime          `json:"now"`
	Events   []Event           `json:"events"`
	Triggers []ScenarioTrigger `json:"triggers"`
}
type ScenarioTrigger struct {
	TriggerId         string  `json:"trigger_id"`
	Type              string  `json:"type"`
	OrderId           string  `json:"order_id"`
	FireAfterStartMin int64   `json:"fire_after_start_min"`
	ExtraDurationMin  *int64  `json:"extra_duration_min,omitempty"`
	TransferredTo     *string `json:"transferred_to,omitempty"`
}
type Health struct {
	Status  string  `json:"status"`
	Team    *string `json:"team,omitempty"`
	Version *string `json:"version,omitempty"`
}
type LoadResult struct {
	Status string           `json:"status"`
	Counts map[string]int64 `json:"counts,omitempty"`
}
type ErrorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Details []struct {
			Field   *string `json:"field,omitempty"`
			Message string  `json:"message"`
		} `json:"details,omitempty"`
	} `json:"error"`
}
