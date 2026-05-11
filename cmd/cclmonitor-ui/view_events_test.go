package main

import (
	"strings"
	"testing"
	"time"

	"github.com/ryosandesu/cclmonitor/internal/eventlog"
)

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
