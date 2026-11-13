package contract

import (
	"encoding/json"
	"magistral/backend/internal/dto"
	"os"
	"testing"
)

func TestBoundary(t *testing.T) {
	v, e := New()
	if e != nil {
		t.Fatal(e)
	}
	raw, e := os.ReadFile("../../testdata/reference.json")
	if e != nil {
		t.Fatal(e)
	}
	var m map[string]any
	json.Unmarshal(raw, &m)
	m["future_field"] = true
	staff := m["staff"].([]any)[0].(map[string]any)
	staff["shift"].(map[string]any)["unknown"] = true
	b, _ := json.Marshal(m)
	var out dto.ReferenceData
	if e = v.Decode(b, "ReferenceData", &out); e != nil {
		t.Fatal(e)
	}
	delete(staff, "height_permit")
	b, _ = json.Marshal(m)
	if e = v.Decode(b, "ReferenceData", &out); e == nil {
		t.Fatal("missing required false boolean accepted")
	}
}
func TestIgnoresFact(t *testing.T) {
	v, _ := New()
	b, _ := os.ReadFile("../../testdata/day.json")
	var m map[string]any
	json.Unmarshal(b, &m)
	m["orders"].([]any)[0].(map[string]any)["fact"] = "ignored"
	b, _ = json.Marshal(m)
	var d dto.DayInput
	if e := v.Decode(b, "DayInput", &d); e != nil {
		t.Fatal(e)
	}
}
