package metrics

import (
	"time"

	"github.com/ryosandesu/cclmonitor/internal/eventlog"
)

// Invocation represents one tool call, produced by pairing a PreToolUse
// pending event with its PostToolUse counterpart via tool_use_id.
type Invocation struct {
	ToolUseID string
	ToolName  string
	Value     string
	StartedAt time.Time
	Outcome   string // executed | denied | cancelled | unknown | untracked | interrupted
	SessionID string
}

// PairInvocations aggregates events into Invocations.
// Pending events within gracePeriod that have no matching PostToolUse are treated as in-flight and excluded.
func PairInvocations(events []eventlog.Event, now time.Time, grace time.Duration) []Invocation {
	type pair struct {
		pre  *eventlog.Event
		post *eventlog.Event
	}
	byID := map[string]*pair{}

	for i := range events {
		e := &events[i]
		p, ok := byID[e.ToolUseID]
		if !ok {
			p = &pair{}
			byID[e.ToolUseID] = p
		}
		switch e.Verdict {
		case "pending":
			p.pre = e
		case "denied":
			// denied is self-contained in PreToolUse and has no pending entry
			p.pre = e
			p.post = e // sentinel: same pointer means "self-contained"
		default:
			p.post = e
		}
	}

	var invs []Invocation
	for _, p := range byID {
		if p.pre == nil {
			continue
		}
		var outcome string
		switch {
		case p.pre.Verdict == "denied":
			outcome = "denied"
		case p.post != nil && p.post != p.pre:
			outcome = p.post.Verdict // executed | unknown | interrupted
		case now.Sub(p.pre.Time) > grace:
			outcome = "cancelled"
		default:
			// in-flight: skip
			continue
		}
		invs = append(invs, Invocation{
			ToolUseID: p.pre.ToolUseID,
			ToolName:  p.pre.ToolName,
			Value:     p.pre.Value,
			StartedAt: p.pre.Time,
			Outcome:   outcome,
			SessionID: p.pre.SessionID,
		})
	}
	return invs
}
