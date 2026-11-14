// Package routing provides road geometry for the UI. It never changes the
// organizer's travel matrix, assignment order, or model clock.
package routing

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

var ErrUnavailable = errors.New("road routing unavailable")
var ErrNoRoute = errors.New("no road route found")

type Point [2]float64 // longitude, latitude
type Geometry struct {
	Type        string  `json:"type"`
	Coordinates []Point `json:"coordinates"`
}
type Leg struct {
	FromSeq     int64    `json:"from_seq"`
	ToSeq       int64    `json:"to_seq"`
	DistanceM   float64  `json:"distance_m"`
	DurationSec float64  `json:"duration_sec"`
	Geometry    Geometry `json:"geometry"`
}
type Path struct {
	Geometry    Geometry `json:"geometry"`
	Legs        []Leg    `json:"legs"`
	DistanceM   float64  `json:"distance_m"`
	DurationSec float64  `json:"duration_sec"`
}
type cacheEntry struct {
	path    Path
	expires time.Time
}
type flight struct {
	done chan struct{}
	path Path
	err  error
}
type Client struct {
	base    string
	http    *http.Client
	mu      sync.Mutex
	cache   map[[32]byte]cacheEntry
	flights map[[32]byte]*flight
	slots   chan struct{}
}

func New(base string) (*Client, error) {
	u, e := url.Parse(strings.TrimRight(base, "/"))
	if e != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return nil, fmt.Errorf("invalid OSRM_URL")
	}
	return &Client{base: u.String(), http: &http.Client{Timeout: 8 * time.Second}, cache: map[[32]byte]cacheEntry{}, flights: map[[32]byte]*flight{}, slots: make(chan struct{}, 4)}, nil
}

// Route preserves point order. Cache keys depend on all coordinates, rather
// than staff IDs or plan versions that can be reused after a dataset reset.
func (c *Client) Route(ctx context.Context, points []Point) (Path, error) {
	if len(points) < 2 || len(points) > 1000 {
		return Path{}, ErrNoRoute
	}
	for _, p := range points {
		if !validPoint(p) {
			return Path{}, ErrNoRoute
		}
	}
	raw, _ := json.Marshal(points)
	key := sha256.Sum256(raw)
	c.mu.Lock()
	if v, ok := c.cache[key]; ok && time.Now().Before(v.expires) {
		c.mu.Unlock()
		return v.path, nil
	}
	if f, ok := c.flights[key]; ok {
		c.mu.Unlock()
		select {
		case <-ctx.Done():
			return Path{}, ctx.Err()
		case <-f.done:
			return f.path, f.err
		}
	}
	f := &flight{done: make(chan struct{})}
	c.flights[key] = f
	c.mu.Unlock()
	p, e := c.fetch(ctx, points)
	c.mu.Lock()
	if e == nil {
		if len(c.cache) >= 64 {
			for k, v := range c.cache {
				delete(c.cache, k)
				if time.Now().Before(v.expires) {
					break
				}
			}
		}
		c.cache[key] = cacheEntry{p, time.Now().Add(15 * time.Minute)}
	}
	f.path, f.err = p, e
	delete(c.flights, key)
	close(f.done)
	c.mu.Unlock()
	return p, e
}
func validPoint(p Point) bool {
	return !math.IsNaN(p[0]) && !math.IsNaN(p[1]) && !math.IsInf(p[0], 0) && !math.IsInf(p[1], 0) && p[0] >= -180 && p[0] <= 180 && p[1] >= -90 && p[1] <= 90
}
func line() Geometry { return Geometry{Type: "LineString", Coordinates: []Point{}} }
func appendPoints(to []Point, from []Point) []Point {
	for _, p := range from {
		if len(to) == 0 || to[len(to)-1] != p {
			to = append(to, p)
		}
	}
	return to
}
func (c *Client) fetch(ctx context.Context, points []Point) (Path, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	select {
	case c.slots <- struct{}{}:
		defer func() { <-c.slots }()
	case <-ctx.Done():
		return Path{}, ctx.Err()
	}
	result := Path{Geometry: line(), Legs: []Leg{}}
	// OSRM's route waypoint limit is commonly 100. Chunks overlap by one point.
	for start := 0; start < len(points)-1; {
		end := min(start+64, len(points))
		parts := make([]string, 0, end-start)
		for _, p := range points[start:end] {
			parts = append(parts, fmt.Sprintf("%.7f,%.7f", p[0], p[1]))
		}
		radiuses := strings.TrimSuffix(strings.Repeat("1000;", len(parts)), ";")
		endpoint := c.base + "/route/v1/driving/" + strings.Join(parts, ";") + "?overview=full&geometries=geojson&steps=true&alternatives=false&generate_hints=false&continue_straight=false&radiuses=" + radiuses
		req, e := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if e != nil {
			return Path{}, ErrUnavailable
		}
		req.Header.Set("User-Agent", "MaxTelecom/0.6 local-road-routing")
		req.Header.Set("Accept", "application/json")
		resp, e := c.http.Do(req)
		if e != nil {
			return Path{}, ErrUnavailable
		}
		raw, e := io.ReadAll(io.LimitReader(resp.Body, 8*1024*1024+1))
		resp.Body.Close()
		if e != nil || len(raw) > 8*1024*1024 {
			return Path{}, ErrUnavailable
		}
		var body struct {
			Code   string `json:"code"`
			Routes []struct {
				Geometry Geometry `json:"geometry"`
				Legs     []struct {
					Duration float64 `json:"duration"`
					Distance float64 `json:"distance"`
					Steps    []struct {
						Geometry Geometry `json:"geometry"`
					} `json:"steps"`
				} `json:"legs"`
			} `json:"routes"`
		}
		if json.Unmarshal(raw, &body) != nil {
			return Path{}, ErrUnavailable
		}
		if body.Code == "NoRoute" || body.Code == "NoSegment" {
			return Path{}, ErrNoRoute
		}
		if resp.StatusCode != 200 || body.Code != "Ok" || len(body.Routes) != 1 {
			return Path{}, ErrUnavailable
		}
		r := body.Routes[0]
		if r.Geometry.Type != "LineString" || len(r.Geometry.Coordinates) < 2 || len(r.Legs) != end-start-1 {
			return Path{}, ErrUnavailable
		}
		for _, p := range r.Geometry.Coordinates {
			if !validPoint(p) {
				return Path{}, ErrUnavailable
			}
		}
		result.Geometry.Coordinates = appendPoints(result.Geometry.Coordinates, r.Geometry.Coordinates)
		for _, l := range r.Legs {
			if l.Distance < 0 || l.Duration < 0 || !finite(l.Distance) || !finite(l.Duration) {
				return Path{}, ErrUnavailable
			}
			leg := Leg{DistanceM: l.Distance, DurationSec: l.Duration, Geometry: line()}
			for _, s := range l.Steps {
				if s.Geometry.Type != "LineString" {
					return Path{}, ErrUnavailable
				}
				for _, p := range s.Geometry.Coordinates {
					if !validPoint(p) {
						return Path{}, ErrUnavailable
					}
				}
				leg.Geometry.Coordinates = appendPoints(leg.Geometry.Coordinates, s.Geometry.Coordinates)
			}
			if l.Distance > 0 && len(leg.Geometry.Coordinates) < 2 {
				return Path{}, ErrUnavailable
			}
			result.Legs = append(result.Legs, leg)
			result.DistanceM += l.Distance
			result.DurationSec += l.Duration
		}
		start = end - 1
	}
	return result, nil
}
func finite(x float64) bool { return !math.IsNaN(x) && !math.IsInf(x, 0) }
