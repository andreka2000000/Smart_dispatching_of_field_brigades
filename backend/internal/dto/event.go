package dto

// EventInput is the validated union's application representation. The OpenAPI
// oneOf validator checks required fields and enums before this is decoded.
type EventInput struct {
	EventId       string    `json:"event_id"`
	Type          EventType `json:"type"`
	Timestamp     DateTime  `json:"timestamp"`
	Order         *Order    `json:"order,omitempty"`
	OrderId       string    `json:"order_id,omitempty"`
	StaffId       string    `json:"staff_id,omitempty"`
	Reason        string    `json:"reason,omitempty"`
	ExpectedEnd   *DateTime `json:"expected_end,omitempty"`
	ActualEnd     *DateTime `json:"actual_end,omitempty"`
	Until         *DateTime `json:"until,omitempty"`
	TransferredTo string    `json:"transferred_to,omitempty"`
}
