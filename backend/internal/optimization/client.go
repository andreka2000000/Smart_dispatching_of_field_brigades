// Package optimization integrates optional, untrusted PyVRP proposals. Exact
// scheduling, validation, scoring and publication remain owned by Go.
package optimization

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	c "magistral/backend/internal/constraints"
	"magistral/backend/internal/dto"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	endpoint string
	http     *http.Client
}
type Route struct {
	StaffID  string   `json:"staff_id"`
	OrderIDs []string `json:"order_ids"`
}
type Candidate struct {
	Model          string  `json:"model"`
	Routes         []Route `json:"routes"`
	ProposedVisits int64   `json:"proposed_visits"`
}
type Response struct {
	Version    string      `json:"version"`
	Seed       int64       `json:"seed"`
	ElapsedMS  int64       `json:"elapsed_ms"`
	Candidates []Candidate `json:"candidates"`
	Reason     string      `json:"reason"`
}

func New(target string) (*Client, error) {
	u, e := url.Parse(target)
	if e != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return nil, fmt.Errorf("OPTIMIZER_URL must be an HTTP(S) base URL")
	}
	return &Client{endpoint: strings.TrimRight(target, "/") + "/solve", http: &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}
func (c *Client) Propose(ctx context.Context, w *c.World, budget time.Duration) (Response, error) {
	excluded := []string{}
	for _, g := range w.Groups {
		excluded = append(excluded, g.OrderIds...)
	}
	payload := struct {
		Reference dto.ReferenceData `json:"reference"`
		Day       dto.DayInput      `json:"day"`
		Matrix    dto.TravelMatrix  `json:"matrix"`
		Excluded  []string          `json:"excluded_order_ids"`
		Budget    float64           `json:"budget_sec"`
		Seed      int64             `json:"seed"`
	}{w.Reference, w.Day, w.Matrix.Data, excluded, budget.Seconds(), 0}
	raw, e := json.Marshal(payload)
	if e != nil {
		return Response{}, e
	}
	request, e := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(raw))
	if e != nil {
		return Response{}, e
	}
	request.Header.Set("Content-Type", "application/json")
	response, e := c.http.Do(request)
	if e != nil {
		return Response{}, fmt.Errorf("optimizer unavailable: %w", e)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return Response{}, fmt.Errorf("optimizer HTTP %d", response.StatusCode)
	}
	raw, e = io.ReadAll(io.LimitReader(response.Body, (1<<20)+1))
	if e != nil || len(raw) > 1<<20 {
		return Response{}, fmt.Errorf("invalid optimizer response size")
	}
	var result Response
	if e = json.Unmarshal(raw, &result); e != nil {
		return result, fmt.Errorf("invalid optimizer JSON")
	}
	if result.Version != "0.14.0" || result.Seed != 0 || len(result.Candidates) > 4 {
		return result, fmt.Errorf("unsupported optimizer response")
	}
	return result, nil
}
