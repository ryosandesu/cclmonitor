package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ryosandesu/cclmonitor/internal/config"
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
	flag.DurationVar(&grace, "grace", 0, "grace period for in-flight pending events (default: config file or 60s)")
	flag.BoolVar(&showVer, "version", false, "print version and exit")
	flag.Parse()

	if showVer {
		fmt.Println("cclmonitor-ui", version)
		os.Exit(0)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, "cannot determine home directory:", err)
		os.Exit(1)
	}
	cfgPath := filepath.Join(home, ".claude", "cclmonitor.yaml")
	logDir = resolveLogDir(logDir, cfgPath, home)
	grace = resolveGrace(grace, cfgPath)

	m := newModel(logDir, grace, snapshot)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// resolveGrace resolves the grace period in priority order:
// --grace flag (> 0) > config grace_sec > default 60s
func resolveGrace(flag time.Duration, cfgPath string) time.Duration {
	if flag > 0 {
		return flag
	}
	if cfg, err := config.LoadFile(cfgPath); err == nil && cfg.EventLog.GraceSec > 0 {
		return time.Duration(cfg.EventLog.GraceSec) * time.Second
	}
	return 60 * time.Second
}

// resolveLogDir resolves the log directory in priority order:
// --logdir flag > config eventlog.logdir > default ~/.claude/
func resolveLogDir(flag, cfgPath, home string) string {
	expandTilde := func(p string) string {
		if strings.HasPrefix(p, "~/") {
			return filepath.Join(home, p[2:])
		}
		return p
	}

	if flag != "" {
		return expandTilde(flag)
	}

	if cfg, err := config.LoadFile(cfgPath); err == nil && cfg.EventLog.LogDir != "" {
		return expandTilde(cfg.EventLog.LogDir)
	}

	return filepath.Join(home, ".claude")
}
