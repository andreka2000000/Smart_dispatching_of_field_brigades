package routing

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

func TestRoadGeometryCacheAndChunking(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Query().Get("steps") != "true" || r.URL.Query().Get("geometries") != "geojson" {
			t.Error("road geometry was not requested")
		}
		coords := []Point{}
		for _, pair := range strings.Split(strings.TrimPrefix(r.URL.Path, "/route/v1/driving/"), ";") {
			parts := strings.Split(pair, ",")
			x, _ := strconv.ParseFloat(parts[0], 64)
			y, _ := strconv.ParseFloat(parts[1], 64)
			coords = append(coords, Point{x, y})
		}
		legs := []any{}
		for i := 1; i < len(coords); i++ {
			legs = append(legs, map[string]any{"duration": 60, "distance": 500, "steps": []any{map[string]any{"geometry": Geometry{"LineString", []Point{coords[i-1], coords[i]}}}}})
		}
		json.NewEncoder(w).Encode(map[string]any{"code": "Ok", "routes": []any{map[string]any{"geometry": Geometry{"LineString", coords}, "legs": legs}}})
	}))
	defer server.Close()
	client, e := New(server.URL)
	if e != nil {
		t.Fatal(e)
	}
	points := []Point{{60.6, 56.8}, {60.7, 56.9}}
	var group sync.WaitGroup
	for i := 0; i < 20; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			p, e := client.Route(context.Background(), points)
			if e != nil || p.DurationSec != 60 || p.DistanceM != 500 || len(p.Geometry.Coordinates) != 2 {
				t.Error("invalid road result", e)
			}
		}()
	}
	group.Wait()
	if calls.Load() != 1 {
		t.Fatalf("duplicate provider calls: %d", calls.Load())
	}
	many := make([]Point, 130)
	for i := range many {
		many[i] = Point{60 + float64(i)*.001, 56}
	}
	p, e := client.Route(context.Background(), many)
	if e != nil || len(p.Legs) != 129 || len(p.Geometry.Coordinates) != 130 || p.DurationSec != 129*60 || calls.Load() != 4 {
		t.Fatalf("chunked route lost legs: %d %d %v", len(p.Legs), calls.Load(), e)
	}
}
func TestRoadFailures(t *testing.T) {
	for i, body := range []string{`{"code":"NoRoute"}`, `{"code":"Ok","routes":[]}`, `not json`} {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, body) }))
			defer s.Close()
			c, _ := New(s.URL)
			_, e := c.Route(context.Background(), []Point{{60, 56}, {61, 57}})
			if e == nil {
				t.Fatal("invalid upstream response accepted")
			}
			if i == 0 && !errors.Is(e, ErrNoRoute) {
				t.Fatal(e)
			}
		})
	}
	if _, e := New("file:///etc/passwd"); e == nil {
		t.Fatal("invalid routing URL accepted")
	}
	c, _ := New("http://127.0.0.1:1")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e := c.Route(ctx, []Point{{60, 56}, {61, 57}}); e == nil {
		t.Fatal("cancelled request succeeded")
	}
}
