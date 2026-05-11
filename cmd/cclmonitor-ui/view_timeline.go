package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func renderTimeline(m model) string {
	var sb strings.Builder
	sb.WriteString(styleMuted.Render("Compliance trend — last 30 days") + "\n")
	sb.WriteString(styleMuted.Render("  ■ ≥90%  ▓ ≥70%  ▒ ≥50%  ░ <50%") + "\n\n")

	barWidth := 24

	for _, b := range m.daily {
		weekday := b.Date.Format("Mon")
		date := b.Date.Format("Jan 02")

		if b.Stats.Compliance < 0 {
			sb.WriteString(fmt.Sprintf("  %s %s  %s\n",
				styleMuted.Render(weekday),
				styleMuted.Render(date),
				styleMuted.Render("· no data"),
			))
			continue
		}

		pct := b.Stats.Compliance * 100
		filled := int(b.Stats.Compliance * float64(barWidth))
		if filled > barWidth {
			filled = barWidth
		}

		var col lipgloss.Color
		switch {
		case pct >= 90:
			col = colorExecuted
		case pct >= 70:
			col = colorCompliance
		case pct >= 50:
			col = colorUnknown
		default:
			col = colorDenied
		}

		bar := lipgloss.NewStyle().Foreground(col).Render(strings.Repeat("█", filled)) +
			styleMuted.Render(strings.Repeat("░", barWidth-filled))

		sb.WriteString(fmt.Sprintf("  %s %s  %s  %.0f%%\n",
			styleMuted.Render(weekday),
			styleHeader.Render(date),
			bar,
			pct,
		))
	}

	sb.WriteString("\n")
	sb.WriteString(renderTabBar(m.activeTab))
	return sb.String()
}
