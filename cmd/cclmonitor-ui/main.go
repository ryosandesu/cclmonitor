package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

var version = "dev"

func main() {
	var (
		logDir   string
		snapshot bool
		grace    time.Duration
		showVer  bool
	)
	flag.StringVar(&logDir, "logdir", "", "JSONL log directory (default: ~/.claude/)")
	flag.BoolVar(&snapshot, "snapshot", false, "one-shot aggregation, no live updates")
	flag.DurationVar(&grace, "grace", 60*time.Second, "grace period for in-flight pending events")
	flag.BoolVar(&showVer, "version", false, "print version and exit")
	flag.Parse()

	if showVer {
		fmt.Println("cclmonitor-ui", version)
		os.Exit(0)
	}

	if logDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintln(os.Stderr, "cannot determine home directory:", err)
			os.Exit(1)
		}
		logDir = filepath.Join(home, ".claude")
	} else if strings.HasPrefix(logDir, "~/") {
		home, _ := os.UserHomeDir()
		logDir = filepath.Join(home, logDir[2:])
	}

	m := newModel(logDir, grace, snapshot)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
