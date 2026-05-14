package main

import (
	"strings"
	"testing"
	"time"

	"github.com/ryosandesu/cclmonitor/internal/eventlog"
)

func TestTruncateValue_BashKeepsHead(t *testing.T) {
	got := truncateValue("Bash", "rm -rf /very/long/path/that/exceeds/limit", 20)
	if !strings.HasPrefix(got, "rm -rf") {
		t.Errorf("Bash truncation should keep head, got: %s", got)
	}
	if len(got) > 20 {
		t.Errorf("length %d > 20", len(got))
	}
}

func TestTruncateValue_EditKeepsTail(t *testing.T) {
	got := truncateValue("Edit", "/Users/ryotakahashi/Desktop/project/src/components/LoginForm.tsx", 20)
	if !strings.HasSuffix(got, "LoginForm.tsx") {
		t.Errorf("Edit truncation should keep filename at tail, got: %s", got)
	}
	if len(got) > 20 {
		t.Errorf("length %d > 20", len(got))
	}
}

func TestTruncateValue_ShortValueUnchanged(t *testing.T) {
	got := truncateValue("Edit", "short.go", 20)
	if got != "short.go" {
		t.Errorf("short value should be unchanged, got: %s", got)
	}
}

func TestRenderEvents_ShowsDateAndTime(t *testing.T) {
	ts := time.Date(2026, 5, 11, 16, 14, 21, 0, time.Local)
	m := model{
		recentEvts: []eventlog.Event{
			{Time: ts, ToolName: "Bash", Value: "ls", Verdict: "executed"},
		},
		height: 20,
	}
	got := renderEvents(m)
	want := "2026-05-11 16:14:21"
	if !strings.Contains(got, want) {
		t.Errorf("renderEvents does not contain date-prefixed timestamp %q\ngot:\n%s", want, got)
	}
}
