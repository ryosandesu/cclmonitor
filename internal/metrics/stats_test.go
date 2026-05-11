package metrics

import (
	"testing"
)

func invocations(outcomes ...string) []Invocation {
	var invs []Invocation
	for i, o := range outcomes {
		invs = append(invs, Invocation{
			ToolUseID: string(rune('a' + i)),
			ToolName:  "Bash",
			Outcome:   o,
		})
	}
	return invs
}

func TestSummarize(t *testing.T) {
	t.Run("compliance formula: executed/(executed+denied+cancelled)", func(t *testing.T) {
		invs := invocations("executed", "executed", "executed", "denied", "cancelled")
		s := Summarize(invs)
		// 3 / (3+1+1) = 0.6
		if s.Compliance < 0.599 || s.Compliance > 0.601 {
			t.Errorf("want 0.6, got %f", s.Compliance)
		}
	})

	t.Run("coverage formula: (executed+denied)/(executed+denied+unknown)", func(t *testing.T) {
		invs := invocations("executed", "denied", "unknown")
		s := Summarize(invs)
		// (1+1)/(1+1+1) = 0.666...
		if s.Coverage < 0.665 || s.Coverage > 0.668 {
			t.Errorf("want ~0.667, got %f", s.Coverage)
		}
	})

	t.Run("N/A when denominator is zero: compliance", func(t *testing.T) {
		invs := invocations("unknown", "interrupted")
		s := Summarize(invs)
		if s.Compliance != -1 {
			t.Errorf("want -1 (N/A), got %f", s.Compliance)
		}
	})

	t.Run("N/A when denominator is zero: coverage", func(t *testing.T) {
		invs := invocations("cancelled", "interrupted")
		s := Summarize(invs)
		if s.Coverage != -1 {
			t.Errorf("want -1 (N/A), got %f", s.Coverage)
		}
	})

	t.Run("counts are accurate", func(t *testing.T) {
		invs := invocations("executed", "executed", "denied", "cancelled", "unknown", "interrupted")
		s := Summarize(invs)
		if s.Executed != 2 {
			t.Errorf("executed: want 2, got %d", s.Executed)
		}
		if s.Denied != 1 {
			t.Errorf("denied: want 1, got %d", s.Denied)
		}
		if s.Cancelled != 1 {
			t.Errorf("cancelled: want 1, got %d", s.Cancelled)
		}
		if s.Unknown != 1 {
			t.Errorf("unknown: want 1, got %d", s.Unknown)
		}
		if s.Interrupted != 1 {
			t.Errorf("interrupted: want 1, got %d", s.Interrupted)
		}
	})
}
