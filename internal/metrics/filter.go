package metrics

import (
	"sort"
	"time"
)

// FilterOpts は Filter の絞り込み条件。
// ゼロ値は「条件なし（全件対象）」を意味する。
type FilterOpts struct {
	From  time.Time
	To    time.Time
	Tools []string // 空なら全ツール
}

// Filter は opts に合致する Invocation だけ返す。
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

// PerTool は ToolName をキーに Stats を算出して返す。
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

// ValueCount は Value ごとの出現回数。
type ValueCount struct {
	Value string
	Count int
}

// TopOffenders は outcomes に該当する Invocation を Value 別に集計し、
// 出現回数降順で上位 limit 件を返す。
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
