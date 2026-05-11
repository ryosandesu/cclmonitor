package metrics

import (
	"testing"
	"time"
)

func invWithTool(outcome, tool string, t time.Time) Invocation {
	return Invocation{ToolName: tool, Outcome: outcome, StartedAt: t}
}

func TestPerTool(t *testing.T) {
	invs := []Invocation{
		{ToolName: "Bash", Outcome: "executed"},
		{ToolName: "Bash", Outcome: "denied"},
		{ToolName: "Edit", Outcome: "executed"},
	}
	result := PerTool(invs)
	if len(result) != 2 {
		t.Fatalf("want 2 tools, got %d", len(result))
	}
	bash := result["Bash"]
	if bash.Executed != 1 || bash.Denied != 1 {
		t.Errorf("Bash: want 1 executed 1 denied, got %+v", bash)
	}
	edit := result["Edit"]
	if edit.Executed != 1 {
		t.Errorf("Edit: want 1 executed, got %+v", edit)
	}
}

func TestTopOffenders(t *testing.T) {
	invs := []Invocation{
		{Value: "rm -rf", Outcome: "denied"},
		{Value: "rm -rf", Outcome: "denied"},
		{Value: "curl|bash", Outcome: "denied"},
		{Value: "npm publish", Outcome: "unknown"},
		{Value: "ls", Outcome: "executed"}, // should not appear
	}
	result := TopOffenders(invs, []string{"denied", "unknown"}, 10)
	if len(result) != 3 {
		t.Fatalf("want 3 offenders, got %d", len(result))
	}
	if result[0].Value != "rm -rf" || result[0].Count != 2 {
		t.Errorf("top offender: want rm -rf x2, got %+v", result[0])
	}
}

func TestTopOffendersLimit(t *testing.T) {
	invs := []Invocation{
		{Value: "a", Outcome: "denied"},
		{Value: "b", Outcome: "denied"},
		{Value: "c", Outcome: "denied"},
	}
	result := TopOffenders(invs, []string{"denied"}, 2)
	if len(result) != 2 {
		t.Fatalf("want 2 (limited), got %d", len(result))
	}
}

func TestFilter(t *testing.T) {
	now := time.Date(2024, 1, 10, 12, 0, 0, 0, time.UTC)
	invs := []Invocation{
		invWithTool("executed", "Bash", now.Add(-1*time.Hour)),
		invWithTool("denied", "Edit", now.Add(-2*time.Hour)),
		invWithTool("executed", "Bash", now.Add(-10*time.Hour)),
	}

	t.Run("filter by time range", func(t *testing.T) {
		opts := FilterOpts{From: now.Add(-3 * time.Hour), To: now}
		result := Filter(invs, opts)
		if len(result) != 2 {
			t.Fatalf("want 2, got %d", len(result))
		}
	})

	t.Run("filter by tool", func(t *testing.T) {
		opts := FilterOpts{Tools: []string{"Edit"}}
		result := Filter(invs, opts)
		if len(result) != 1 || result[0].ToolName != "Edit" {
			t.Errorf("want 1 Edit, got %v", result)
		}
	})

	t.Run("empty opts returns all", func(t *testing.T) {
		result := Filter(invs, FilterOpts{})
		if len(result) != 3 {
			t.Fatalf("want 3, got %d", len(result))
		}
	})
}
