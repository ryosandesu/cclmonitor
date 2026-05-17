package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/ryosandesu/cclmonitor/internal/metrics"
)

func renderOverview(m model) string {
	cardW := (m.width - 6) / 2
	if cardW < 20 {
		cardW = 20
	}

	left := renderComplianceCard(m.stats, cardW)
	right := renderCoverageCard(m.stats, cardW)
	cards := lipgloss.JoinHorizontal(lipgloss.Top, left, "  ", right)

	perToolSection := renderPerToolBars(m)
	violationsSection := renderRecentViolations(m)
	bottomRow := lipgloss.JoinHorizontal(lipgloss.Top,
		perToolSection,
		strings.Repeat(" ", 4),
		violationsSection,
	)

	tabs := renderTabBar(m.activeTab)

	return cards + "\n\n" + bottomRow + "\n\n" + tabs
}

func renderComplianceCard(s metrics.Stats, width int) string {
	title := styleCardTitle.Render("Claude Compliance")
	score := formatScore(s.Compliance, colorCompliance)
	counts := fmt.Sprintf(
		"  executed:  %d\n  %s%d%s\n  %s%d%s",
		s.Executed,
		styleDenied.Render("denied:    "), s.Denied, styleDenied.Render("  ✗"),
		styleCancelled.Render("cancelled: "), s.Cancelled, styleCancelled.Render("  ✗"),
	)
	body := title + "\n\n" + score + "\n\n" + counts
	return styleCard.Width(width).Render(body)
}

func renderCoverageCard(s metrics.Stats, width int) string {
	title := styleCardTitle.Render("Rule Coverage")
	score := formatScore(s.Coverage, colorCoverage)
	hint := ""
	if s.Coverage != -1 && s.Coverage < 0.8 {
		hint = "\n  " + styleUnknown.Render("→ consider adding rules")
	}
	counts := fmt.Sprintf("  unknown: %d", s.Unknown) + hint
	body := title + "\n\n" + score + "\n\n" + counts
	return styleCard.Width(width).Render(body)
}

func formatScore(v float64, color lipgloss.Color) string {
	if v < 0 {
		return styleScore.Foreground(colorMuted).Render("  N/A")
	}
	pct := v * 100
	var col lipgloss.Color
	switch {
	case pct >= 90:
		col = colorExecuted
	case pct >= 70:
		col = color
	default:
		col = colorDenied
	}
	return styleScore.Foreground(col).Render(fmt.Sprintf("  %.1f %%", pct))
}

func renderPerToolBars(m model) string {
	tools := []string{"Bash", "Edit", "Write", "Read"}
	var sb strings.Builder
	sb.WriteString(styleMuted.Render("Per-Tool Compliance") + "\n")
	for _, tool := range tools {
		s, ok := m.perTool[tool]
		if !ok {
			sb.WriteString(fmt.Sprintf("  %-6s %s\n", tool, styleMuted.Render("no data")))
			continue
		}
		bar := complianceBar(s.Compliance, 16)
		pct := "N/A"
		if s.Compliance >= 0 {
			pct = fmt.Sprintf("%.1f%%", s.Compliance*100)
		}
		sb.WriteString(fmt.Sprintf("  %-6s %s %s\n", tool, bar, pct))
	}
	return sb.String()
}

func complianceBar(v float64, width int) string {
	if v < 0 {
		return strings.Repeat("░", width)
	}
	filled := int(v * float64(width))
	if filled > width {
		filled = width
	}
	return styleExecuted.Render(strings.Repeat("█", filled)) +
		styleMuted.Render(strings.Repeat("░", width-filled))
}

func renderRecentViolations(m model) string {
	var sb strings.Builder
	sb.WriteString(styleMuted.Render("Recent Violations") + "\n")

	invs := make([]metrics.Invocation, len(m.invocations))
	copy(invs, m.invocations)
	sort.Slice(invs, func(i, j int) bool {
		return invs[i].StartedAt.After(invs[j].StartedAt)
	})

	shown := 0
	for _, inv := range invs {
		if shown >= 5 {
			break
		}
		if inv.Outcome != "denied" && inv.Outcome != "cancelled" && inv.Outcome != "unknown" {
			continue
		}
		ts := inv.StartedAt.Local().Format("01/02 15:04")
		val := truncateValue(inv.ToolName, inv.Value, 20)
		line := fmt.Sprintf("  %s %s %-6s %s",
			styleMuted.Render(ts),
			verdictStyle(inv.Outcome).Render(fmt.Sprintf("%-9s", inv.Outcome)),
			inv.ToolName,
			val,
		)
		sb.WriteString(line + "\n")
		shown++
	}
	if shown == 0 {
		sb.WriteString(styleMuted.Render("  (none)") + "\n")
	}
	return sb.String()
}

func renderTabBar(active tab) string {
	tabs := []struct {
		label string
		t     tab
	}{
		{"[1] Overview", tabOverview},
		{"[2] Tools", tabTools},
		{"[3] Timeline", tabTimeline},
		{"[4] Events", tabEvents},
	}
	var parts []string
	for _, tb := range tabs {
		if tb.t == active {
			parts = append(parts, styleTabActive.Render(tb.label))
		} else {
			parts = append(parts, styleTabInactive.Render(tb.label))
		}
	}
	return styleMuted.Render("──── ") + strings.Join(parts, styleMuted.Render("  ")) + styleMuted.Render(" ────")
}
