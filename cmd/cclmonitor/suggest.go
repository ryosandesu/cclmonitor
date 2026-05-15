package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ryosandesu/cclmonitor/internal/claudelog"
	"github.com/ryosandesu/cclmonitor/internal/config"
	"github.com/ryosandesu/cclmonitor/internal/configfile"
	"github.com/ryosandesu/cclmonitor/internal/defaults"
	"github.com/ryosandesu/cclmonitor/internal/eventlog"
	"github.com/ryosandesu/cclmonitor/internal/suggest"
)

const (
	defaultSuggestDays    = 30
	defaultSuggestMin     = 5
	defaultSuggestThresh  = 10
	defaultSuggestTarget  = "global"
)

type suggestOpts struct {
	Days               int
	MinCount           int
	Target             string // absolute path to cclmonitor.yaml
	InsufficientThresh int
	DryRun             bool
	LogDir             string
	CWD                string
	HomeDir            string
	Now                time.Time
}

func runSuggestCmd(args []string) int {
	fs := flag.NewFlagSet("suggest", flag.ExitOnError)
	days := fs.Int("days", defaultSuggestDays, "lookback window in days")
	minCount := fs.Int("min-count", defaultSuggestMin, "minimum hits required for a suggestion")
	target := fs.String("target", defaultSuggestTarget, "write target: global or project")
	thresh := fs.Int("insufficient-threshold", defaultSuggestThresh, "below this event count, fall back to defaults mode")
	dryRun := fs.Bool("dry-run", false, "do not write; show only")
	_ = fs.Parse(args)

	cwd := mustGetwd()
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, "could not resolve home dir:", err)
		return 1
	}

	targetPath, err := resolveTargetPath(*target, cwd, home)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	opts := suggestOpts{
		Days:               *days,
		MinCount:           *minCount,
		Target:             targetPath,
		InsufficientThresh: *thresh,
		DryRun:             *dryRun,
		LogDir:             resolveLogDir(globalCfgPath()),
		CWD:                cwd,
		HomeDir:            home,
		Now:                time.Now(),
	}
	return runSuggest(os.Stdin, os.Stdout, opts)
}

func runSuggest(in io.Reader, out io.Writer, opts suggestOpts) int {
	reader := bufio.NewReader(in)

	from := opts.Now.AddDate(0, 0, -opts.Days)
	to := opts.Now

	events, source := selectEvents(opts, from, to)

	fmt.Fprintf(out, "Scanning %s (%s to %s)...\n", source, from.Format("2006-01-02"), to.Format("2006-01-02"))
	fmt.Fprintf(out, "Found %d events.\n", len(events))
	fmt.Fprintf(out, "Target: %s\n", opts.Target)

	if len(events) < opts.InsufficientThresh {
		fmt.Fprintf(out, "Insufficient data (< %d events).\n\n", opts.InsufficientThresh)
		return applyDefaults(reader, out, opts)
	}

	suggestions := suggest.Aggregate(events, opts.CWD, opts.MinCount)
	if len(suggestions) == 0 {
		fmt.Fprintln(out, "No suggestions met --min-count threshold.")
		return 0
	}

	current, _ := config.LoadFile(opts.Target)

	fmt.Fprintf(out, "Found %d suggestions.\n\n", len(suggestions))
	return applySuggestions(reader, out, opts, suggestions, current)
}

func selectEvents(opts suggestOpts, from, to time.Time) ([]eventlog.Event, string) {
	// Primary: cclmonitor logs
	events, err := eventlog.ReadRange(opts.LogDir, from, to)
	if err == nil && len(events) >= opts.InsufficientThresh {
		return events, "cclmonitor logs"
	}

	// Fallback: Claude transcripts for this project
	projDir := filepath.Join(opts.HomeDir, ".claude", "projects", claudelog.EncodeCwd(opts.CWD))
	claudeEvents, cerr := claudelog.ReadProjectTranscripts(projDir, from, to)
	if cerr == nil && len(claudeEvents) >= opts.InsufficientThresh {
		return claudeEvents, "Claude transcripts (allow suggestions only — no verdict info)"
	}

	// Return whichever has the most data; the caller compares to the threshold.
	if len(events) >= len(claudeEvents) {
		return events, "cclmonitor logs"
	}
	return claudeEvents, "Claude transcripts"
}

func applySuggestions(reader *bufio.Reader, out io.Writer, opts suggestOpts, suggestions []suggest.Suggestion, current *config.Config) int {
	backupDone := false
	added, skipped, alreadyPresent := 0, 0, 0

	for i, s := range suggestions {
		header := fmt.Sprintf("[%d/%d] %s.%s", i+1, len(suggestions), s.Tool, s.Section)
		if suggest.IsDuplicate(current, s) {
			label := "already in config"
			if s.Section == "deny" {
				label = "already denied"
			}
			fmt.Fprintf(out, "%s\n  %s: %s\n  hits:  %d (%s — no action)\n\n", header, s.Kind, s.Pattern, s.Count, label)
			alreadyPresent++
			continue
		}

		fmt.Fprintf(out, "%s\n  %s: %s\n  hits:  %d\n", header, s.Kind, s.Pattern, s.Count)

		if opts.DryRun {
			fmt.Fprintf(out, "  (dry-run: not writing)\n\n")
			continue
		}

		choice, err := promptYNQ(reader, out, "  Add to config? [y/N/q]: ")
		if err != nil {
			fmt.Fprintf(out, "  read error: %v\n", err)
			return 1
		}
		switch choice {
		case 'y':
			if !backupDone {
				bak, berr := configfile.Backup(opts.Target)
				if berr != nil {
					fmt.Fprintf(out, "  backup failed: %v\n", berr)
					return 1
				}
				if bak != "" {
					fmt.Fprintf(out, "  Backup: %s\n", bak)
				}
				backupDone = true
			}
			if err := configfile.Insert(opts.Target, s.Tool, s.Section, s.Kind, s.Pattern); err != nil {
				fmt.Fprintf(out, "  write failed: %v\n", err)
				return 1
			}
			added++
			fmt.Fprintln(out, "  ✓ Added.")
		case 'q':
			fmt.Fprintf(out, "\nStopping. %d added, %d skipped, %d already present.\n", added, skipped, alreadyPresent)
			return 0
		default:
			skipped++
			fmt.Fprintln(out, "  Skipped.")
		}
		fmt.Fprintln(out)
	}
	fmt.Fprintf(out, "Done. %d added, %d skipped, %d already present.\n", added, skipped, alreadyPresent)
	return 0
}

func applyDefaults(reader *bufio.Reader, out io.Writer, opts suggestOpts) int {
	fmt.Fprintln(out, "Apply baseline security defaults? This adds:")
	fmt.Fprintln(out, "  - secrets:      block .env / .pem / .key / *credentials* / .ssh")
	fmt.Fprintln(out, "  - shell-safety: block sudo / rm -rf / curl | sh")
	fmt.Fprintln(out, "  - git-safety:   block push --force / reset --hard / clean -f")

	if opts.DryRun {
		fmt.Fprintln(out, "(dry-run: not writing)")
		return 0
	}

	choice, err := promptYNQ(reader, out, "Apply all? [y/N]: ")
	if err != nil {
		fmt.Fprintf(out, "read error: %v\n", err)
		return 1
	}
	if choice != 'y' {
		fmt.Fprintln(out, "Skipped.")
		return 0
	}

	bak, berr := configfile.Backup(opts.Target)
	if berr != nil {
		fmt.Fprintf(out, "backup failed: %v\n", berr)
		return 1
	}
	if bak != "" {
		fmt.Fprintf(out, "Backup: %s\n", bak)
	}

	written := 0
	cfg := defaults.Builtin()
	for tool, tr := range cfg.Rules {
		for _, r := range tr.Allow {
			if err := insertRule(opts.Target, tool, "allow", r); err == nil {
				written++
			}
		}
		for _, r := range tr.Deny {
			if err := insertRule(opts.Target, tool, "deny", r); err == nil {
				written++
			}
		}
	}
	fmt.Fprintf(out, "✓ Applied %d baseline rules.\n", written)
	return 0
}

func insertRule(path, tool, section string, r config.Rule) error {
	kind := "regex"
	pattern := r.Regex
	if r.Glob != "" {
		kind = "glob"
		pattern = r.Glob
	}
	return configfile.Insert(path, tool, section, kind, pattern)
}

func promptYNQ(in io.Reader, out io.Writer, msg string) (rune, error) {
	reader, ok := in.(*bufio.Reader)
	if !ok {
		reader = bufio.NewReader(in)
	}
	for {
		fmt.Fprint(out, msg)
		line, err := reader.ReadString('\n')
		if err != nil && line == "" {
			return 0, err
		}
		ans := strings.ToLower(strings.TrimSpace(line))
		switch ans {
		case "y", "yes":
			return 'y', nil
		case "q", "quit":
			return 'q', nil
		case "", "n", "no":
			return 'n', nil
		}
	}
}

func resolveTargetPath(target, cwd, home string) (string, error) {
	switch target {
	case "global":
		return filepath.Join(home, ".claude", "cclmonitor.yaml"), nil
	case "project":
		return filepath.Join(cwd, ".claude", "cclmonitor.yaml"), nil
	default:
		return "", fmt.Errorf("invalid --target %q (must be global or project)", target)
	}
}

func resolveLogDir(globalCfgPath string) string {
	cfg, err := config.LoadFile(globalCfgPath)
	if err != nil || cfg == nil {
		return defaultLogDir()
	}
	if cfg.EventLog.LogDir == "" {
		return defaultLogDir()
	}
	dir := cfg.EventLog.LogDir
	if strings.HasPrefix(dir, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			dir = filepath.Join(home, dir[2:])
		}
	}
	return dir
}

func defaultLogDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return filepath.Join(home, ".claude")
}
