// Package insights derives incidents and auditable warnings from model time.
// It has no network, database, wall-clock or AI dependencies.
package insights

import (
	"crypto/sha256"
	"fmt"
	c "magistral/backend/internal/constraints"
	"magistral/backend/internal/dto"
	"sort"
	"time"
)

func id(prefix, key string) string {
	return fmt.Sprintf("%s-%x", prefix, sha256.Sum256([]byte(key)))[:len(prefix)+1+20]
}
func ptr[T any](v T) *T { return &v }

// Detect partitions orders into deterministic bounded 60-minute windows per node.
// Confirmed active groups survive shrinking to two members; a singleton is
// released to scheduling because the contract forbids one-member PGP groups.
// History preserves group identity and first detection across cancellation,
// growth, merging, and a later recurrence of the same incident.
func Detect(w *c.World, active []dto.PgpGroup, history map[string]dto.PgpGroup, cancelled map[string]dto.CancelledOrder, transferred, pinned map[string]bool) []dto.PgpGroup {
	if history == nil {
		history = map[string]dto.PgpGroup{}
	}
	started := map[string]bool{}
	if w.Previous != nil {
		for _, r := range w.Previous.Routes {
			for _, s := range r.Stops {
				if s.OrderId != nil && (c.Time(s.Start).Before(c.Time(w.Day.Now)) || pinned[*s.OrderId]) {
					started[*s.OrderId] = true
				}
			}
		}
	}
	nodes := map[string][]dto.Order{}
	eligible := map[string]dto.Order{}
	for _, o := range w.Day.Orders {
		_, gone := cancelled[o.OrderId]
		if gone || transferred[o.OrderId] || started[o.OrderId] || !w.Work[o.WorkTypeCode].PgpRelevant || c.Time(o.RegisteredAt).After(c.Time(w.Day.Now)) {
			continue
		}
		nodes[o.AccessNodeId] = append(nodes[o.AccessNodeId], o)
		eligible[o.OrderId] = o
	}
	for _, orders := range nodes {
		sort.Slice(orders, func(i, j int) bool {
			if !c.Time(orders[i].RegisteredAt).Equal(c.Time(orders[j].RegisteredAt)) {
				return c.Time(orders[i].RegisteredAt).Before(c.Time(orders[j].RegisteredAt))
			}
			return orders[i].OrderId < orders[j].OrderId
		})
	}
	clusters := map[string][]string{}
	taken := map[string]bool{}
	active = append([]dto.PgpGroup{}, active...)
	sort.Slice(active, func(i, j int) bool { return active[i].GroupId < active[j].GroupId })
	for _, g := range active {
		ids := []string{}
		for _, x := range g.OrderIds {
			if o, ok := eligible[x]; ok && o.AccessNodeId == g.AccessNodeId {
				ids = append(ids, x)
			}
		}
		if len(ids) < 2 {
			continue
		}
		for _, x := range ids {
			taken[x] = true
		}
		first, last := c.Time(eligible[ids[0]].RegisteredAt), c.Time(eligible[ids[0]].RegisteredAt)
		for _, x := range ids {
			t := c.Time(eligible[x].RegisteredAt)
			if t.Before(first) {
				first = t
			}
			if t.After(last) {
				last = t
			}
		}
		for _, o := range nodes[g.AccessNodeId] {
			if taken[o.OrderId] {
				continue
			}
			a, b := first, last
			t := c.Time(o.RegisteredAt)
			if t.Before(a) {
				a = t
			}
			if t.After(b) {
				b = t
			}
			if b.Sub(a) <= time.Hour {
				ids = append(ids, o.OrderId)
				taken[o.OrderId] = true
				first = a
				last = b
			}
		}
		clusters[g.GroupId] = ids
	}
	for node, all := range nodes {
		orders := []dto.Order{}
		for _, o := range all {
			if !taken[o.OrderId] {
				orders = append(orders, o)
			}
		}
		for left := 0; left < len(orders); {
			right := left
			for right < len(orders) && c.Time(orders[right].RegisteredAt).Sub(c.Time(orders[left].RegisteredAt)) <= time.Hour {
				right++
			}
			if right-left < 3 {
				left++
				continue
			}
			ids := []string{}
			for _, o := range orders[left:right] {
				ids = append(ids, o.OrderId)
			}
			clusters[node+"|"+orders[left].OrderId] = ids
			left = right
		}
	}
	prior := make([]dto.PgpGroup, 0, len(history)+len(active))
	seen := map[string]bool{}
	for _, g := range active {
		history[g.GroupId] = g
	}
	for _, g := range history {
		prior = append(prior, g)
	}
	sort.Slice(prior, func(i, j int) bool {
		if prior[i].DetectedAt != prior[j].DetectedAt {
			return c.Time(prior[i].DetectedAt).Before(c.Time(prior[j].DetectedAt))
		}
		return prior[i].GroupId < prior[j].GroupId
	})
	out := []dto.PgpGroup{}
	keys := make([]string, 0, len(clusters))
	for key := range clusters {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		members := clusters[key]
		sort.Strings(members)
		node := eligible[members[0]].AccessNodeId
		g := dto.PgpGroup{GroupId: id("PGP", w.Day.Date+"|"+node+"|"+members[0]), AccessNodeId: node, OrderIds: members, DetectedAt: w.Day.Now, Action: "TRANSFER_TO_STATION"}
		for _, old := range prior {
			if old.AccessNodeId != node || seen[old.GroupId] {
				continue
			}
			overlap := false
			for _, x := range members {
				if c.Has(old.OrderIds, x) {
					overlap = true
					break
				}
			}
			if overlap {
				g.GroupId = old.GroupId
				g.DetectedAt = old.DetectedAt
				break
			}
		}
		seen[g.GroupId] = true
		all := map[string]bool{}
		if h, ok := history[g.GroupId]; ok {
			for _, x := range h.OrderIds {
				all[x] = true
			}
		}
		for _, x := range members {
			all[x] = true
		}
		h := g
		h.OrderIds = []string{}
		for x := range all {
			h.OrderIds = append(h.OrderIds, x)
		}
		sort.Strings(h.OrderIds)
		history[g.GroupId] = h
		out = append(out, g)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].GroupId < out[j].GroupId })
	return out
}
