package domain

import (
	"magistral/backend/internal/dto"
	"testing"
)

func TestClock(t *testing.T) {
	c, e := NewClock("2026-10-26T03:30:17Z")
	if e != nil || c.Now.Format("15:04:05") != "08:30:17" {
		t.Fatal(c, e)
	}
	if _, e = c.Advance("2026-10-26T08:30:16+05:00"); e == nil {
		t.Fatal("accepted backwards time")
	}
	if _, e = c.Advance("2026-10-26T08:30:17+05:00"); e != nil {
		t.Fatal(e)
	}
	if _, e = NewClock("2026-10-26T08:30:17.5+05:00"); e == nil {
		t.Fatal("fractional seconds accepted")
	}
}
func TestAsymmetricMatrix(t *testing.T) {
	m, e := NewMatrix(dto.TravelMatrix{Points: []string{"a", "b"}, DurationsSec: [][]int64{{0, 17}, {32, 0}}, DistancesM: [][]int64{{0, 100}, {200, 0}}})
	if e != nil {
		t.Fatal(e)
	}
	a, _, _ := m.Travel("a", "b")
	b, _, _ := m.Travel("b", "a")
	if a != 17 || b != 32 {
		t.Fatal("matrix symmetrized")
	}
	if _, _, e = m.Travel("missing", "a"); e == nil {
		t.Fatal("unknown point accepted")
	}
}
