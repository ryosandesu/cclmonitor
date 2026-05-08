package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ryosandesu/cclmonitor/internal/config"
	"github.com/ryosandesu/cclmonitor/internal/hookio"
	"github.com/ryosandesu/cclmonitor/internal/match"
	"github.com/ryosandesu/cclmonitor/internal/notify"
	"github.com/ryosandesu/cclmonitor/internal/store"
)

func run(r io.Reader, globalCfgPath string) int {
	payload, err := hookio.Parse(r)
	if err != nil {
		return 0
	}

	value, err := hookio.MatchValue(payload)
	if err != nil {
		return 0
	}

	cfg := loadMergedConfig(globalCfgPath, payload.Cwd)
	cfg = config.ExpandCwd(cfg, payload.Cwd)

	rules := cfg.Rules[payload.ToolName]
	verdict, err := match.Evaluate(rules, value)
	if err != nil {
		return 0
	}

	event := notify.Event{
		Time:      time.Now(),
		SessionID: payload.SessionID,
		ToolName:  payload.ToolName,
		Value:     value,
		Verdict:   verdictString(verdict),
	}

	switch verdict {
	case match.Deny:
		if !checkDuplicate(cfg.Notify, event) {
			_ = notify.Notify(cfg.Notify, event)
		}
		return 2
	case match.Allow:
		if cfg.Mode == "dev" && !checkDuplicate(cfg.Notify, event) {
			_ = notify.Notify(cfg.Notify, event)
		}
		return 0
	default:
		if !checkDuplicate(cfg.Notify, event) {
			_ = notify.Notify(cfg.Notify, event)
		}
		return 0
	}
}

// checkDuplicate returns true if the event is a duplicate within the dedup window.
// Returns false (not duplicate) on any error so notifications are not silently lost.
func checkDuplicate(cfg config.NotifyConfig, event notify.Event) bool {
	if cfg.DedupWindowSec <= 0 {
		return false
	}
	dbPath := resolveDBPath(cfg.DBDir)
	s, err := store.Open(dbPath)
	if err != nil {
		return false
	}
	defer s.Close()

	hash := inputHash(event.ToolName, event.Value)
	dup, err := s.IsDuplicate(event.SessionID, hash, cfg.DedupWindowSec)
	if err != nil {
		return false
	}
	return dup
}

func inputHash(toolName, value string) string {
	sum := sha256.Sum256([]byte(toolName + "|" + value))
	return fmt.Sprintf("%x", sum)
}

func resolveDBPath(dbDir string) string {
	dir := dbDir
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "cclmonitor.db"
		}
		dir = filepath.Join(home, ".claude")
	} else if strings.HasPrefix(dir, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return filepath.Join(dir[2:], "cclmonitor.db")
		}
		dir = filepath.Join(home, dir[2:])
	}
	_ = os.MkdirAll(dir, 0755)
	return filepath.Join(dir, "cclmonitor.db")
}

func loadMergedConfig(globalPath, cwd string) *config.Config {
	global, err := config.LoadFile(globalPath)
	if err != nil {
		global = &config.Config{}
	}

	projectPath := cwd + "/.claude/cclmonitor.yaml"
	project, err := config.LoadFile(projectPath)
	if err != nil {
		return global
	}

	return config.Merge(global, project)
}

func verdictString(v match.Verdict) string {
	switch v {
	case match.Deny:
		return "deny"
	case match.Allow:
		return "allow"
	default:
		return "unknown"
	}
}
