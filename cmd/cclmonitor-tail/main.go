package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ryosandesu/cclmonitor/internal/config"
	"github.com/ryosandesu/cclmonitor/internal/eventlog"
)

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
)

func verdictColor(verdict string) string {
	switch verdict {
	case "allow":
		return colorGreen
	case "deny":
		return colorRed
	case "unknown":
		return colorYellow
	default:
		return colorReset
	}
}

func FormatLine(e eventlog.Event) string {
	color := verdictColor(e.Verdict)
	ts := e.Time.UTC().Format("15:04:05")
	verdict := fmt.Sprintf("%-7s", e.Verdict)
	return fmt.Sprintf("%s%s [%s] %s: %s%s", color, ts, verdict, e.ToolName, e.Value, colorReset)
}

func logFilePath(dir string, t time.Time) string {
	return filepath.Join(dir, "cclmonitor."+t.Format("2006-01-02")+".log")
}

func resolveDir(dir string) string {
	if dir == "" {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".claude")
	}
	if strings.HasPrefix(dir, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, dir[2:])
	}
	return dir
}

func tail(logDir string, out io.Writer) error {
	dir := resolveDir(logDir)
	path := logFilePath(dir, time.Now())

	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("log file not found: %s", path)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		printLine(scanner.Text(), out)
	}

	for {
		time.Sleep(500 * time.Millisecond)
		for scanner.Scan() {
			printLine(scanner.Text(), out)
		}
	}
}

func printLine(raw string, out io.Writer) {
	var e eventlog.Event
	if err := json.Unmarshal([]byte(raw), &e); err != nil {
		fmt.Fprintln(out, raw)
		return
	}
	fmt.Fprintln(out, FormatLine(e))
}

func globalCfgPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".claude", "cclmonitor.yaml")
}

func main() {
	logDir := ""
	if cfg, err := config.LoadFile(globalCfgPath()); err == nil {
		logDir = cfg.EventLog.LogDir
	}
	if err := tail(logDir, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
