package main

import (
	"strings"
	"testing"
	"time"

	"github.com/ryosandesu/cclmonitor/internal/eventlog"
)

func TestFormatLine(t *testing.T) {
	ts := time.Date(2024, 1, 15, 14, 32, 1, 0, time.UTC)

	tests := []struct {
		event  eventlog.Event
		checks []string
	}{
		{
			event:  eventlog.Event{Time: ts, ToolName: "Bash", Value: "ls -la", Verdict: "executed"},
			checks: []string{"14:32:01", "executed", "Bash", "ls -la", colorGreen},
		},
		{
			event:  eventlog.Event{Time: ts, ToolName: "Edit", Value: "/path/to/file", Verdict: "denied"},
			checks: []string{"14:32:01", "denied", "Edit", "/path/to/file", colorRed},
		},
		{
			event:  eventlog.Event{Time: ts, ToolName: "Write", Value: "/tmp/out", Verdict: "unknown"},
			checks: []string{"14:32:01", "unknown", "Write", "/tmp/out", colorYellow},
		},
		{
			event:  eventlog.Event{Time: ts, ToolName: "Bash", Value: "sleep 60", Verdict: "interrupted"},
			checks: []string{"14:32:01", "interrupted", "Bash", "sleep 60", colorCyan},
		},
		{
			event:  eventlog.Event{Time: ts, ToolName: "Bash", Value: "git status", Verdict: "pending"},
			checks: []string{"14:32:01", "pending", "Bash", "git status", colorBlue},
		},
	}

	for _, tc := range tests {
		line := FormatLine(tc.event)
		for _, check := range tc.checks {
			if !strings.Contains(line, check) {
				t.Errorf("FormatLine(%+v): want %q in output %q", tc.event, check, line)
			}
		}
	}
}

func TestVerdictColor(t *testing.T) {
	cases := map[string]string{
		"executed":    colorGreen,
		"denied":      colorRed,
		"unknown":     colorYellow,
		"interrupted": colorCyan,
		"pending":     colorBlue,
		"other":       colorReset,
	}
	for verdict, want := range cases {
		got := verdictColor(verdict)
		if got != want {
			t.Errorf("verdictColor(%q) = %q, want %q", verdict, got, want)
		}
	}
}
