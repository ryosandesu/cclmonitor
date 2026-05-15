package eventlog

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeTestLog(t *testing.T, dir string, date time.Time, events []Event) {
	t.Helper()
	path := filepath.Join(dir, "cclmonitor."+date.Format("2006-01-02")+".log")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	for _, e := range events {
		b, _ := json.Marshal(e)
		fmt.Fprintf(f, "%s\n", b)
	}
}

func TestTruncateDay(t *testing.T) {
	loc := time.FixedZone("JST", 9*60*60)
	in := time.Date(2024, 1, 15, 14, 30, 45, 0, loc)
	want := time.Date(2024, 1, 15, 0, 0, 0, 0, loc)
	got := TruncateDay(in)
	if !got.Equal(want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestReadRange(t *testing.T) {
	dir := t.TempDir()
	base := time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC)

	day0Events := []Event{
		{Time: base.Add(1 * time.Hour), ToolName: "Bash", Value: "ls", Verdict: "executed"},
		{Time: base.Add(2 * time.Hour), ToolName: "Edit", Value: "foo.go", Verdict: "denied"},
	}
	day1Events := []Event{
		{Time: base.Add(25 * time.Hour), ToolName: "Write", Value: "bar.go", Verdict: "executed"},
		{Time: base.Add(26 * time.Hour), ToolName: "Read", Value: "baz.go", Verdict: "unknown"},
	}
	day2Events := []Event{
		{Time: base.Add(49 * time.Hour), ToolName: "Bash", Value: "pwd", Verdict: "pending"},
	}

	writeTestLog(t, dir, base, day0Events)
	writeTestLog(t, dir, base.Add(24*time.Hour), day1Events)
	writeTestLog(t, dir, base.Add(48*time.Hour), day2Events)

	t.Run("returns events within range", func(t *testing.T) {
		from := base
		to := base.Add(48 * time.Hour)
		events, err := ReadRange(dir, from, to)
		if err != nil {
			t.Fatal(err)
		}
		if len(events) != 4 {
			t.Fatalf("want 4 events, got %d", len(events))
		}
	})

	t.Run("excludes events outside range", func(t *testing.T) {
		from := base.Add(24 * time.Hour)
		to := base.Add(48 * time.Hour)
		events, err := ReadRange(dir, from, to)
		if err != nil {
			t.Fatal(err)
		}
		if len(events) != 2 {
			t.Fatalf("want 2 events, got %d", len(events))
		}
		if events[0].ToolName != "Write" {
			t.Errorf("want Write, got %s", events[0].ToolName)
		}
	})

	t.Run("skips missing file silently", func(t *testing.T) {
		from := base.Add(72 * time.Hour)
		to := base.Add(96 * time.Hour)
		events, err := ReadRange(dir, from, to)
		if err != nil {
			t.Fatal(err)
		}
		if len(events) != 0 {
			t.Fatalf("want 0 events, got %d", len(events))
		}
	})

	t.Run("skips corrupt JSON lines", func(t *testing.T) {
		corruptDir := t.TempDir()
		path := filepath.Join(corruptDir, "cclmonitor.2024-01-10.log")
		f, _ := os.Create(path)
		fmt.Fprintln(f, `{"time":"2024-01-10T01:00:00Z","tool_name":"Bash","verdict":"executed"}`)
		fmt.Fprintln(f, `NOT VALID JSON`)
		fmt.Fprintln(f, `{"time":"2024-01-10T02:00:00Z","tool_name":"Edit","verdict":"denied"}`)
		f.Close()

		from := time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC)
		to := time.Date(2024, 1, 11, 0, 0, 0, 0, time.UTC)
		events, err := ReadRange(corruptDir, from, to)
		if err != nil {
			t.Fatal(err)
		}
		if len(events) != 2 {
			t.Fatalf("want 2 events (corrupt line skipped), got %d", len(events))
		}
	})
}

func TestReaderPoll(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")

	e1 := Event{Time: time.Now(), ToolName: "Bash", Value: "ls", Verdict: "executed"}
	b1, _ := json.Marshal(e1)
	os.WriteFile(path, append(b1, '\n'), 0600)

	r, err := NewReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	t.Run("first poll returns all existing events", func(t *testing.T) {
		events, err := r.Poll()
		if err != nil {
			t.Fatal(err)
		}
		if len(events) != 1 {
			t.Fatalf("want 1, got %d", len(events))
		}
	})

	t.Run("second poll returns empty when no new events", func(t *testing.T) {
		events, err := r.Poll()
		if err != nil {
			t.Fatal(err)
		}
		if len(events) != 0 {
			t.Fatalf("want 0, got %d", len(events))
		}
	})

	t.Run("poll returns only new events after append", func(t *testing.T) {
		e2 := Event{Time: time.Now(), ToolName: "Edit", Value: "foo.go", Verdict: "denied"}
		b2, _ := json.Marshal(e2)
		f, _ := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0600)
		fmt.Fprintf(f, "%s\n", b2)
		f.Close()

		events, err := r.Poll()
		if err != nil {
			t.Fatal(err)
		}
		if len(events) != 1 {
			t.Fatalf("want 1 new event, got %d", len(events))
		}
		if events[0].ToolName != "Edit" {
			t.Errorf("want Edit, got %s", events[0].ToolName)
		}
	})
}
