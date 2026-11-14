package domain

import "magistral/backend/internal/dto"

type Unavailability struct {
	From  dto.DateTime  `json:"from"`
	Until *dto.DateTime `json:"until"`
}

// Facts are persisted alongside the current day, never derived from wall time.
type EventState struct {
	EndOverrides map[string]dto.DateTime     `json:"end_overrides"`
	NewOrderAt   map[string]dto.DateTime     `json:"new_order_at"`
	Unavailable  map[string][]Unavailability `json:"unavailable"`
	Transferred  map[string]bool             `json:"transferred"`
	PgpHistory   map[string]dto.PgpGroup     `json:"pgp_history"`
}

func (s *EventState) Init() {
	if s.PgpHistory == nil {
		s.PgpHistory = map[string]dto.PgpGroup{}
	}
	if s.Transferred == nil {
		s.Transferred = map[string]bool{}
	}
	if s.EndOverrides == nil {
		s.EndOverrides = map[string]dto.DateTime{}
	}
	if s.NewOrderAt == nil {
		s.NewOrderAt = map[string]dto.DateTime{}
	}
	if s.Unavailable == nil {
		s.Unavailable = map[string][]Unavailability{}
	}
}
