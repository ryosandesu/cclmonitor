package metrics

import (
	"testing"
	"time"

	"github.com/ryosandesu/cclmonitor/internal/eventlog"
)

var baseTime = time.Date(2024, 1, 10, 12, 0, 0, 0, time.UTC)

func event(verdict, toolName, toolUseID string, offset time.Duration) eventlog.Event {
	return eventlog.Event{
		Time:      baseTime.Add(offset),
		SessionID: "sess1",
		ToolUseID: toolUseID,
		ToolName:  toolName,
		Value:     toolName + "-value",
		Verdict:   verdict,
	}
}

func TestPairInvocations(t *testing.T) {
	grace := 60 * time.Second
	now := baseTime.Add(10 * time.Minute)

	tests := []struct {
		name        string
		events      []eventlog.Event
		wantOutcome string
		wantCount   int
	}{
		{
			name: "denied single → denied",
			events: []eventlog.Event{
				event("denied", "Bash", "id1", 0),
			},
			wantOutcome: "denied",
			wantCount:   1,
		},
		{
			name: "pending + executed → executed",
			events: []eventlog.Event{
				event("pending", "Bash", "id2", 0),
				event("executed", "Bash", "id2", time.Second),
			},
			wantOutcome: "executed",
			wantCount:   1,
		},
		{
			name: "pending + unknown → unknown",
			events: []eventlog.Event{
				event("pending", "Edit", "id3", 0),
				event("unknown", "Edit", "id3", time.Second),
			},
			wantOutcome: "unknown",
			wantCount:   1,
		},
		{
			name: "pending + interrupted → interrupted",
			events: []eventlog.Event{
				event("pending", "Write", "id4", 0),
				event("interrupted", "Write", "id4", time.Second),
			},
			wantOutcome: "interrupted",
			wantCount:   1,
		},
		{
			name: "old pending without post → cancelled",
			events: []eventlog.Event{
				// 10 minutes ago, well past gracePeriod of 60s
				event("pending", "Bash", "id5", -10*time.Minute),
			},
			wantOutcome: "cancelled",
			wantCount:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			invs := PairInvocations(tt.events, now, grace)
			if len(invs) != tt.wantCount {
				t.Fatalf("want %d invocations, got %d", tt.wantCount, len(invs))
			}
			if invs[0].Outcome != tt.wantOutcome {
				t.Errorf("want outcome %q, got %q", tt.wantOutcome, invs[0].Outcome)
			}
		})
	}

	t.Run("recent pending without post → in-flight, excluded", func(t *testing.T) {
		events := []eventlog.Event{
			// 30s before now (now = baseTime+10min), within gracePeriod of 60s
			event("pending", "Bash", "id6", 10*time.Minute-30*time.Second),
		}
		invs := PairInvocations(events, now, grace)
		if len(invs) != 0 {
			t.Fatalf("want 0 (in-flight excluded), got %d", len(invs))
		}
	})

	t.Run("order-independent: post before pre", func(t *testing.T) {
		events := []eventlog.Event{
			event("executed", "Bash", "id7", time.Second),
			event("pending", "Bash", "id7", 0),
		}
		invs := PairInvocations(events, now, grace)
		if len(invs) != 1 || invs[0].Outcome != "executed" {
			t.Errorf("want executed, got %v", invs)
		}
	})
}
