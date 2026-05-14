package suggest

import (
	"testing"
	"time"

	"github.com/ryosandesu/cclmonitor/internal/eventlog"
)

func bashEvent(value, verdict string) eventlog.Event {
	return eventlog.Event{
		Time:     time.Now(),
		ToolName: "Bash",
		Value:    value,
		Verdict:  verdict,
	}
}

func fileEvent(tool, path, verdict string) eventlog.Event {
	return eventlog.Event{
		Time:     time.Now(),
		ToolName: tool,
		Value:    path,
		Verdict:  verdict,
	}
}

func TestAggregateBash_AllowFromUnknown(t *testing.T) {
	events := []eventlog.Event{
		bashEvent("pnpm install", "unknown"),
		bashEvent("pnpm install --frozen-lockfile", "unknown"),
		bashEvent("pnpm install foo", "unknown"),
		bashEvent("docker ps", "unknown"),
		bashEvent("docker ps -a", "unknown"),
		bashEvent("git status", "executed"), // wrong verdict
	}

	got := Aggregate(events, "/cwd", 2)

	// Expect: pnpm install (3 hits), docker ps (2 hits)
	want := []Suggestion{
		{Tool: "Bash", Section: "allow", Kind: "regex", Pattern: `(^|[\s;&|])pnpm\s+install\b`, Count: 3},
		{Tool: "Bash", Section: "allow", Kind: "regex", Pattern: `(^|[\s;&|])docker\s+ps\b`, Count: 2},
	}
	if !equalSuggestions(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestAggregateBash_DenyFromDenied(t *testing.T) {
	events := []eventlog.Event{
		bashEvent("npm install", "denied"),
		bashEvent("npm install foo", "denied"),
		bashEvent("npm install bar", "denied"),
		bashEvent("yarn add x", "denied"),
		bashEvent("yarn add y", "denied"),
	}

	got := Aggregate(events, "/cwd", 2)

	want := []Suggestion{
		{Tool: "Bash", Section: "deny", Kind: "regex", Pattern: `\bnpm\s+install\b`, Count: 3},
		{Tool: "Bash", Section: "deny", Kind: "regex", Pattern: `\byarn\s+add\b`, Count: 2},
	}
	if !equalSuggestions(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestAggregateFiles_AllowFromUnknown(t *testing.T) {
	events := []eventlog.Event{
		fileEvent("Edit", "/cwd/src/a.ts", "unknown"),
		fileEvent("Edit", "/cwd/src/b.ts", "unknown"),
		fileEvent("Edit", "/cwd/src/components/c.tsx", "unknown"),
		fileEvent("Read", "/cwd/docs/x.md", "unknown"),
	}

	got := Aggregate(events, "/cwd", 1)

	// Expect Edit suggestion for ts (2 hits in src), tsx (1 hit), Read for md (1 hit)
	// With minCount=1, all should appear. Order: by count desc.
	want := []Suggestion{
		{Tool: "Edit", Section: "allow", Kind: "glob", Pattern: "<cwd>/src/**/*.ts", Count: 2},
		{Tool: "Edit", Section: "allow", Kind: "glob", Pattern: "<cwd>/src/**/*.tsx", Count: 1},
		{Tool: "Read", Section: "allow", Kind: "glob", Pattern: "<cwd>/docs/**/*.md", Count: 1},
	}
	if !equalSuggestions(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestAggregate_BelowMinCountFiltered(t *testing.T) {
	events := []eventlog.Event{
		bashEvent("pnpm install", "unknown"),
		bashEvent("pnpm install", "unknown"),
		bashEvent("docker ps", "unknown"), // only 1 hit
	}

	got := Aggregate(events, "/cwd", 2)
	if len(got) != 1 {
		t.Fatalf("got %d suggestions, want 1", len(got))
	}
	if got[0].Pattern != `(^|[\s;&|])pnpm\s+install\b` {
		t.Errorf("expected pnpm install, got %q", got[0].Pattern)
	}
}

func TestAggregate_IgnoresUnsupportedVerdicts(t *testing.T) {
	events := []eventlog.Event{
		bashEvent("pnpm install", "executed"),
		bashEvent("pnpm install", "pending"),
		bashEvent("pnpm install", "cancelled"),
		bashEvent("pnpm install", "interrupted"),
	}
	got := Aggregate(events, "/cwd", 1)
	if len(got) != 0 {
		t.Errorf("expected 0 suggestions, got %d", len(got))
	}
}

func TestAggregate_EmptyEvents(t *testing.T) {
	got := Aggregate(nil, "/cwd", 1)
	if got != nil && len(got) != 0 {
		t.Errorf("expected nil/empty, got %+v", got)
	}
}

func TestAggregate_SortedByCountDesc(t *testing.T) {
	events := []eventlog.Event{
		bashEvent("a x", "unknown"),
		bashEvent("b y", "unknown"),
		bashEvent("b y", "unknown"),
		bashEvent("b y", "unknown"),
		bashEvent("c z", "unknown"),
		bashEvent("c z", "unknown"),
	}
	got := Aggregate(events, "/cwd", 1)
	if len(got) != 3 {
		t.Fatalf("expected 3 suggestions, got %d", len(got))
	}
	if got[0].Count < got[1].Count || got[1].Count < got[2].Count {
		t.Errorf("not sorted desc: %+v", got)
	}
}

func equalSuggestions(a, b []Suggestion) bool {
	if len(a) != len(b) {
		return false
	}
	// Compare as sets keyed by (Tool, Section, Pattern), then verify Count and Kind.
	bm := make(map[string]Suggestion)
	for _, s := range b {
		bm[s.Tool+"|"+s.Section+"|"+s.Pattern] = s
	}
	for _, s := range a {
		match, ok := bm[s.Tool+"|"+s.Section+"|"+s.Pattern]
		if !ok || match.Count != s.Count || match.Kind != s.Kind {
			return false
		}
	}
	return true
}
