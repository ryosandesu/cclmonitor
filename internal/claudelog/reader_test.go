package claudelog

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReadProjectTranscripts_ExtractsToolUse(t *testing.T) {
	dir := t.TempDir()
	jsonl := `{"type":"permission-mode","permissionMode":"default"}
{"type":"user","message":{"content":"hello"}}
{"type":"assistant","timestamp":"2026-05-10T10:00:00Z","message":{"content":[{"type":"text","text":"ok"},{"type":"tool_use","id":"toolu_1","name":"Bash","input":{"command":"pnpm install"}}]}}
{"type":"assistant","timestamp":"2026-05-10T10:01:00Z","message":{"content":[{"type":"tool_use","id":"toolu_2","name":"Read","input":{"file_path":"/proj/src/app.ts"}}]}}
{"type":"assistant","timestamp":"2026-05-10T10:02:00Z","message":{"content":[{"type":"tool_use","id":"toolu_3","name":"Edit","input":{"file_path":"/proj/src/app.ts","old_string":"a","new_string":"b"}}]}}
{"type":"assistant","timestamp":"2026-05-10T10:03:00Z","message":{"content":[{"type":"tool_use","id":"toolu_4","name":"Grep","input":{"pattern":"foo"}}]}}
`
	path := filepath.Join(dir, "session.jsonl")
	if err := os.WriteFile(path, []byte(jsonl), 0644); err != nil {
		t.Fatal(err)
	}

	from := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	events, err := ReadProjectTranscripts(dir, from, to)
	if err != nil {
		t.Fatal(err)
	}

	// Should pick up Bash/Read/Edit but skip Grep (not a tool we track).
	if len(events) != 3 {
		t.Fatalf("got %d events, want 3 (events=%+v)", len(events), events)
	}

	wantTools := map[string]string{
		"toolu_1": "Bash",
		"toolu_2": "Read",
		"toolu_3": "Edit",
	}
	for _, e := range events {
		if wantTools[e.ToolUseID] != e.ToolName {
			t.Errorf("tool_use %s: got %s, want %s", e.ToolUseID, e.ToolName, wantTools[e.ToolUseID])
		}
		if e.Verdict != "unknown" {
			t.Errorf("expected verdict=unknown, got %s", e.Verdict)
		}
	}
}

func TestReadProjectTranscripts_FiltersByTime(t *testing.T) {
	dir := t.TempDir()
	jsonl := `{"type":"assistant","timestamp":"2026-04-01T10:00:00Z","message":{"content":[{"type":"tool_use","id":"old","name":"Bash","input":{"command":"old cmd"}}]}}
{"type":"assistant","timestamp":"2026-05-10T10:00:00Z","message":{"content":[{"type":"tool_use","id":"new","name":"Bash","input":{"command":"new cmd"}}]}}
`
	path := filepath.Join(dir, "session.jsonl")
	if err := os.WriteFile(path, []byte(jsonl), 0644); err != nil {
		t.Fatal(err)
	}

	from := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	events, err := ReadProjectTranscripts(dir, from, to)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	if events[0].ToolUseID != "new" {
		t.Errorf("got %s, want new", events[0].ToolUseID)
	}
}

func TestReadProjectTranscripts_MissingDir(t *testing.T) {
	events, err := ReadProjectTranscripts("/nonexistent/path", time.Time{}, time.Now())
	if err != nil {
		t.Errorf("expected no error for missing dir, got %v", err)
	}
	if len(events) != 0 {
		t.Errorf("got %d events, want 0", len(events))
	}
}

func TestEncodeCwdToProjectDirName(t *testing.T) {
	got := EncodeCwd("/Users/foo/Desktop/proj")
	want := "-Users-foo-Desktop-proj"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
