package main

import (
	"io"
	"time"

	"github.com/ryosandesu/cclmonitor/internal/config"
	"github.com/ryosandesu/cclmonitor/internal/eventlog"
	"github.com/ryosandesu/cclmonitor/internal/hookio"
	"github.com/ryosandesu/cclmonitor/internal/match"
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

	if verdict == match.Deny {
		_ = eventlog.Write(cfg.EventLog, eventlog.Event{
			Time:      time.Now(),
			SessionID: payload.SessionID,
			ToolName:  payload.ToolName,
			Value:     value,
			Verdict:   "denied",
		})
		return 2
	}
	return 0
}

func runPost(r io.Reader, globalCfgPath string) int {
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

	// deny はPreToolUseでブロック済みのためPostToolUseには届かない
	if verdict == match.Deny {
		return 0
	}

	v := "executed"
	if verdict == match.Unknown {
		v = "unknown"
	}

	_ = eventlog.Write(cfg.EventLog, eventlog.Event{
		Time:      time.Now(),
		SessionID: payload.SessionID,
		ToolName:  payload.ToolName,
		Value:     value,
		Verdict:   v,
	})
	return 0
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
