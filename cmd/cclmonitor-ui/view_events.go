package main

import (
	"fmt"
	"strings"
)

func renderEvents(m model) string {
	var sb strings.Builder
	sb.WriteString(styleHeader.Render(fmt.Sprintf("%-10s %-12s %-8s %s", "Time", "Verdict", "Tool", "Value")) + "\n")
	sb.WriteString(styleMuted.Render(strings.Repeat("─", 60)) + "\n")

	evts := m.recentEvts
	start := len(evts) - 1 - m.eventsOffset
	shown := 0
	maxRows := m.height - 8
	if maxRows < 5 {
		maxRows = 5
	}

	for i := start; i >= 0 && shown < maxRows; i-- {
		e := evts[i]
		ts := e.Time.Local().Format("15:04:05")
		val := e.Value
		if len(val) > 40 {
			val = val[:37] + "..."
		}
		line := fmt.Sprintf("%-10s %-12s %-8s %s",
			ts,
			verdictStyle(e.Verdict).Render(e.Verdict),
			e.ToolName,
			val,
		)
		sb.WriteString(line + "\n")
		shown++
	}

	if shown == 0 {
		sb.WriteString(styleMuted.Render("  (no events)") + "\n")
	}

	sb.WriteString("\n")
	sb.WriteString(styleMuted.Render(fmt.Sprintf("  j/k to scroll  total: %d events", len(evts))) + "\n")
	sb.WriteString("\n")
	sb.WriteString(renderTabBar(m.activeTab))
	return sb.String()
}
