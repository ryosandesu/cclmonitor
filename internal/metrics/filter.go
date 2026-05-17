package metrics

import (
	"sort"
	"time"
)

// FilterOpts specifies the criteria for Filter.
// Zero values mean "no restriction (all events)".
type FilterOpts struct {
	From  time.Time
	To    time.Time
	Tools []string // empty means all tools
}

// Filter returns only the Invocations that match opts.
func Filter(invs []Invocation, opts FilterOpts) []Invocation {
	toolSet := make(map[string]bool, len(opts.Tools))
	for _, t := range opts.Tools {
		toolSet[t] = true
	}

	var result []Invocation
	for _, inv := range invs {
		if !opts.From.IsZero() && inv.StartedAt.Before(opts.From) {
			continue
		}
		if !opts.To.IsZero() && !inv.StartedAt.Before(opts.To) {
			continue
		}
		if len(toolSet) > 0 && !toolSet[inv.ToolName] {
			continue
		}
		result = append(result, inv)
	}
	return result
}

// PerTool returns a Stats map keyed by ToolName.
func PerTool(invs []Invocation) map[string]Stats {
	groups := map[string][]Invocation{}
	for _, inv := range invs {
		groups[inv.ToolName] = append(groups[inv.ToolName], inv)
	}
	result := make(map[string]Stats, len(groups))
	for tool, group := range groups {
		result[tool] = Summarize(group)
	}
	return result
}

// ValueCount holds the occurrence count for a distinct Value.
type ValueCount struct {
	Value string
	Count int
}

// TopOffenders aggregates Invocations with the given outcomes by Value
// and returns up to limit entries sorted by count descending.
func TopOffenders(invs []Invocation, outcomes []string, limit int) []ValueCount {
	outcomeSet := make(map[string]bool, len(outcomes))
	for _, o := range outcomes {
		outcomeSet[o] = true
	}

	counts := map[string]int{}
	for _, inv := range invs {
		if outcomeSet[inv.Outcome] {
			counts[inv.Value]++
		}
	}

	result := make([]ValueCount, 0, len(counts))
	for v, c := range counts {
		result = append(result, ValueCount{Value: v, Count: c})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Count != result[j].Count {
			return result[i].Count > result[j].Count
		}
		return result[i].Value < result[j].Value
	})
	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}
	return result
}
