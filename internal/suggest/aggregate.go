package suggest

import (
	"sort"

	"github.com/ryosandesu/cclmonitor/internal/eventlog"
)

// Suggestion describes a rule that the suggest engine proposes adding
// to cclmonitor.yaml. Section is "allow" or "deny"; Kind is "regex" or "glob".
type Suggestion struct {
	Tool    string
	Section string
	Kind    string
	Pattern string
	Count   int
}

// Aggregate processes events and returns suggestions sorted by descending count.
// Only events with verdict "unknown" (allow candidates) or "denied" (deny candidates)
// contribute. cwd is used to compute <cwd>-relative globs for file-based tools.
// Suggestions with Count < minCount are filtered out.
func Aggregate(events []eventlog.Event, cwd string, minCount int) []Suggestion {
	type key struct {
		tool    string
		section string
		kind    string
		pattern string
	}
	counts := make(map[key]int)

	for _, e := range events {
		section := sectionForVerdict(e.Verdict)
		if section == "" {
			continue
		}
		switch e.ToolName {
		case "Bash":
			bk, ok := ExtractBashKey(e.Value)
			if !ok {
				continue
			}
			k := key{tool: "Bash", section: section, kind: "regex", pattern: BashKeyToRegex(bk, section)}
			counts[k]++
		case "Edit", "Write", "Read":
			gl, ok := ExtractFileGlob(cwd, e.Value)
			if !ok {
				continue
			}
			k := key{tool: e.ToolName, section: section, kind: "glob", pattern: gl}
			counts[k]++
		}
	}

	out := make([]Suggestion, 0, len(counts))
	for k, c := range counts {
		if c < minCount {
			continue
		}
		out = append(out, Suggestion{
			Tool:    k.tool,
			Section: k.section,
			Kind:    k.kind,
			Pattern: k.pattern,
			Count:   c,
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Pattern < out[j].Pattern
	})
	return out
}

func sectionForVerdict(verdict string) string {
	switch verdict {
	case "unknown":
		return "allow"
	case "denied":
		return "deny"
	default:
		return ""
	}
}
