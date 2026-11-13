package domain

import (
	"fmt"
	"magistral/backend/internal/dto"
	"time"
)

var Location = time.FixedZone("Asia/Yekaterinburg", 5*3600)

// ModelClock never reads the wall clock. Future event processing must persist
// Advance's result in the same transaction as the event and plan.
type ModelClock struct{ Now time.Time }

func NewClock(value string) (ModelClock, error) {
	t, e := time.Parse(time.RFC3339, value)
	if e != nil || t.Nanosecond() != 0 {
		return ModelClock{}, fmt.Errorf("expected ISO 8601 with offset and whole seconds")
	}
	return ModelClock{t.In(Location)}, nil
}
func (c ModelClock) Advance(value string) (ModelClock, error) {
	next, e := NewClock(value)
	if e != nil {
		return c, e
	}
	if next.Now.Before(c.Now) {
		return c, fmt.Errorf("model time cannot move backwards")
	}
	return next, nil
}

type Matrix struct {
	Data  dto.TravelMatrix
	Index map[string]int
}

func NewMatrix(data dto.TravelMatrix) (*Matrix, error) {
	m := &Matrix{data, make(map[string]int, len(data.Points))}
	n := len(data.Points)
	if len(data.DurationsSec) != n || len(data.DistancesM) != n {
		return nil, fmt.Errorf("matrices must be N x N")
	}
	for i, id := range data.Points {
		if id == "" {
			return nil, fmt.Errorf("empty point ID")
		}
		if _, ok := m.Index[id]; ok {
			return nil, fmt.Errorf("duplicate point %s", id)
		}
		m.Index[id] = i
		if len(data.DurationsSec[i]) != n || len(data.DistancesM[i]) != n {
			return nil, fmt.Errorf("matrices must be N x N")
		}
		for j := 0; j < n; j++ {
			if data.DurationsSec[i][j] < 0 || data.DistancesM[i][j] < 0 {
				return nil, fmt.Errorf("negative matrix entry")
			}
		}
		if data.DurationsSec[i][i] != 0 || data.DistancesM[i][i] != 0 {
			return nil, fmt.Errorf("matrix diagonal must be zero")
		}
	}
	return m, nil
}
func (m *Matrix) Travel(from, to string) (int64, int64, error) {
	i, ok := m.Index[from]
	j, ok2 := m.Index[to]
	if !ok || !ok2 {
		return 0, 0, fmt.Errorf("unknown location")
	}
	return m.Data.DurationsSec[i][j], m.Data.DistancesM[i][j], nil
}
