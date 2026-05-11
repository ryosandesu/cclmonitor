package main

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Tab1     key.Binding
	Tab2     key.Binding
	Tab3     key.Binding
	Tab4     key.Binding
	PeriodT  key.Binding
	Period7  key.Binding
	PeriodM  key.Binding
	PeriodA  key.Binding
	Up       key.Binding
	Down     key.Binding
	Refresh  key.Binding
	Pause    key.Binding
	Help     key.Binding
	Quit     key.Binding
}

var keys = keyMap{
	Tab1:    key.NewBinding(key.WithKeys("1"), key.WithHelp("1", "overview")),
	Tab2:    key.NewBinding(key.WithKeys("2"), key.WithHelp("2", "tools")),
	Tab3:    key.NewBinding(key.WithKeys("3"), key.WithHelp("3", "timeline")),
	Tab4:    key.NewBinding(key.WithKeys("4"), key.WithHelp("4", "events")),
	PeriodT: key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "today")),
	Period7: key.NewBinding(key.WithKeys("7"), key.WithHelp("7", "7d")),
	PeriodM: key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "30d")),
	PeriodA: key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "all")),
	Up:      key.NewBinding(key.WithKeys("k", "up"), key.WithHelp("k/↑", "up")),
	Down:    key.NewBinding(key.WithKeys("j", "down"), key.WithHelp("j/↓", "down")),
	Refresh: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
	Pause:   key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "pause/resume")),
	Help:    key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
	Quit:    key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
}
