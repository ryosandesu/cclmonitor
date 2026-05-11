package main

import (
	"fmt"
	"strings"

	"github.com/ryosandesu/cclmonitor/internal/metrics"
)

func renderTools(m model) string {
	tools := []string{"Bash", "Edit", "Write", "Read"}
	var sb strings.Builder
	for _, tool := range tools {
		s, ok := m.perTool[tool]
		if !ok {
			s = metrics.Stats{Compliance: -1, Coverage: -1}
		}
		pct := "N/A"
		if s.Compliance >= 0 {
			pct = fmt.Sprintf("%.1f%%", s.Compliance*100)
		}
		sb.WriteString(fmt.Sprintf("%s  %s compliance\n", tool, pct))
		sb.WriteString(renderOutcomeRow("executed", s.Executed, maxCount(s)))
		sb.WriteString(renderOutcomeRow("denied", s.Denied, maxCount(s)))
		sb.WriteString(renderOutcomeRow("cancelled", s.Cancelled, maxCount(s)))
		sb.WriteString(renderOutcomeRow("unknown", s.Unknown, maxCount(s)))
		sb.WriteString(renderOutcomeRow("interrupted", s.Interrupted, maxCount(s)))
		sb.WriteString("\n")
	}

	sb.WriteString(renderTabBar(m.activeTab))
	return sb.String()
}

func renderOutcomeRow(outcome string, count, max int) string {
	barLen := 16
	filled := 0
	if max > 0 {
		filled = count * barLen / max
	}
	bar := verdictStyle(outcome).Render(strings.Repeat("█", filled)) +
		styleMuted.Render(strings.Repeat("░", barLen-filled))
	suffix := ""
	if outcome == "denied" || outcome == "cancelled" {
		suffix = "  " + styleDenied.Render("✗")
	}
	return fmt.Sprintf("  %-12s %s %3d%s\n", outcome, bar, count, suffix)
}

func maxCount(s metrics.Stats) int {
	vals := []int{s.Executed, s.Denied, s.Cancelled, s.Unknown, s.Interrupted}
	m := 0
	for _, v := range vals {
		if v > m {
			m = v
		}
	}
	return m
}
