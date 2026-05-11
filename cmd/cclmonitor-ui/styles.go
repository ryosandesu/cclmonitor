package main

import "github.com/charmbracelet/lipgloss"

var (
	colorCompliance = lipgloss.Color("#00d7af") // teal
	colorCoverage   = lipgloss.Color("#5f87ff") // blue
	colorDenied     = lipgloss.Color("#ff5f5f") // red
	colorExecuted   = lipgloss.Color("#5fff87") // green
	colorUnknown    = lipgloss.Color("#ffff5f") // yellow
	colorCancelled  = lipgloss.Color("#ff875f") // orange
	colorInterrupt  = lipgloss.Color("#5fffff") // cyan
	colorMuted      = lipgloss.Color("#626262") // gray
	colorTitle      = lipgloss.Color("#ffffff")

	styleCard = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#444444")).
			Padding(0, 1)

	styleCardTitle = lipgloss.NewStyle().
			Foreground(colorMuted).
			Bold(false)

	styleScore = lipgloss.NewStyle().
			Bold(true).
			Padding(0, 1)

	styleTabActive = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorTitle).
			Underline(true)

	styleTabInactive = lipgloss.NewStyle().
				Foreground(colorMuted)

	styleHeader = lipgloss.NewStyle().
			Foreground(colorMuted)

	styleDenied    = lipgloss.NewStyle().Foreground(colorDenied)
	styleExecuted  = lipgloss.NewStyle().Foreground(colorExecuted)
	styleUnknown   = lipgloss.NewStyle().Foreground(colorUnknown)
	styleCancelled = lipgloss.NewStyle().Foreground(colorCancelled)
	styleInterrupt = lipgloss.NewStyle().Foreground(colorInterrupt)
	styleMuted     = lipgloss.NewStyle().Foreground(colorMuted)
)

func verdictStyle(verdict string) lipgloss.Style {
	switch verdict {
	case "executed":
		return styleExecuted
	case "denied":
		return styleDenied
	case "cancelled":
		return styleCancelled
	case "unknown":
		return styleUnknown
	case "interrupted":
		return styleInterrupt
	default:
		return styleMuted
	}
}
