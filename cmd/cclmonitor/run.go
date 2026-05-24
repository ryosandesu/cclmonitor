package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/ryosandesu/cclmonitor/internal/config"
	"github.com/ryosandesu/cclmonitor/internal/eventlog"
	"github.com/ryosandesu/cclmonitor/internal/hookio"
	"github.com/ryosandesu/cclmonitor/internal/match"
)

func run(r io.Reader, w io.Writer, globalCfgPath string) int {
	payload, err := hookio.Parse(r)
	if err != nil {
		return 0
	}
	value, matchErr := hookio.MatchValue(payload)
	if matchErr != nil && !errors.Is(matchErr, hookio.ErrUnsupportedTool) {
		return 0
	}

	cfg := loadMergedConfig(globalCfgPath, payload.Cwd)
	cfg = config.ExpandCwd(cfg, payload.Cwd)

	if errors.Is(matchErr, hookio.ErrUnsupportedTool) {
		_ = eventlog.Write(cfg.EventLog, eventlog.Event{
			Time:      time.Now(),
			SessionID: payload.SessionID,
			ToolUseID: payload.ToolUseID,
			ToolName:  payload.ToolName,
			Verdict:   "pending",
		})
		return 0
	}

	rules := cfg.Rules[payload.ToolName]

	var verdict match.Verdict
	if payload.ToolName == "Bash" {
		tokens := match.SplitBashCommands(value)
		verdict, err = evaluateTokens(rules, tokens)
	} else {
		verdict, err = match.Evaluate(rules, value)
	}
	if err != nil {
		return 0
	}

	// default_verdict: deny blocks unknown commands but logs them as "unknown"
	// so Coverage and Unknown counts remain meaningful in the UI
	blockedByDefault := verdict == match.Unknown && cfg.DefaultVerdict == "deny"

	if verdict == match.Deny {
		_ = eventlog.Write(cfg.EventLog, eventlog.Event{
			Time:      time.Now(),
			SessionID: payload.SessionID,
			ToolUseID: payload.ToolUseID,
			ToolName:  payload.ToolName,
			Value:     value,
			Verdict:   "denied",
		})
		writeBlockReason(w, payload.ToolName, value)
		return 2
	}

	if blockedByDefault {
		_ = eventlog.Write(cfg.EventLog, eventlog.Event{
			Time:      time.Now(),
			SessionID: payload.SessionID,
			ToolUseID: payload.ToolUseID,
			ToolName:  payload.ToolName,
			Value:     value,
			Verdict:   "unknown",
		})
		writeBlockReason(w, payload.ToolName, value)
		return 2
	}

	_ = eventlog.Write(cfg.EventLog, eventlog.Event{
		Time:      time.Now(),
		SessionID: payload.SessionID,
		ToolUseID: payload.ToolUseID,
		ToolName:  payload.ToolName,
		Value:     value,
		Verdict:   "pending",
	})
	return 0
}

// evaluateTokens evaluates each token against rules.
// One Deny short-circuits to Deny; Unknown demotes overall from Allow to Unknown.
func evaluateTokens(rules config.ToolRules, tokens []string) (match.Verdict, error) {
	overall := match.Allow
	for _, token := range tokens {
		if token == "" {
			continue
		}
		v, err := match.Evaluate(rules, token)
		if err != nil {
			return match.Unknown, err
		}
		if v == match.Deny {
			return match.Deny, nil
		}
		if v == match.Unknown {
			overall = match.Unknown
		}
	}
	return overall, nil
}

func runPost(r io.Reader, globalCfgPath string) int {
	payload, err := hookio.Parse(r)
	if err != nil {
		return 0
	}
	value, matchErr := hookio.MatchValue(payload)
	if matchErr != nil && !errors.Is(matchErr, hookio.ErrUnsupportedTool) {
		return 0
	}

	cfg := loadMergedConfig(globalCfgPath, payload.Cwd)
	cfg = config.ExpandCwd(cfg, payload.Cwd)

	if errors.Is(matchErr, hookio.ErrUnsupportedTool) {
		_ = eventlog.Write(cfg.EventLog, eventlog.Event{
			Time:      time.Now(),
			SessionID: payload.SessionID,
			ToolUseID: payload.ToolUseID,
			ToolName:  payload.ToolName,
			Verdict:   "untracked",
		})
		return 0
	}

	rules := cfg.Rules[payload.ToolName]

	var verdict match.Verdict
	if payload.ToolName == "Bash" {
		tokens := match.SplitBashCommands(value)
		verdict, err = evaluateTokens(rules, tokens)
	} else {
		verdict, err = match.Evaluate(rules, value)
	}
	if err != nil {
		return 0
	}

	// deny is blocked in PreToolUse and never reaches PostToolUse
	if verdict == match.Deny {
		return 0
	}

	if payload.ToolResponse.Interrupted {
		_ = eventlog.Write(cfg.EventLog, eventlog.Event{
			Time:      time.Now(),
			SessionID: payload.SessionID,
			ToolUseID: payload.ToolUseID,
			ToolName:  payload.ToolName,
			Value:     value,
			Verdict:   "interrupted",
		})
		return 0
	}

	v := "executed"
	if verdict == match.Unknown {
		v = "unknown"
	}

	_ = eventlog.Write(cfg.EventLog, eventlog.Event{
		Time:      time.Now(),
		SessionID: payload.SessionID,
		ToolUseID: payload.ToolUseID,
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

	projectPath := filepath.Join(cwd, ".claude", "cclmonitor.yaml")
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

func writeBlockReason(w io.Writer, toolName, value string) {
	msg := fmt.Sprintf(
		"cclmonitor POLICY BLOCK: %s %q is denied by policy. Do not attempt workarounds or alternative approaches. Report this violation to the user and stop.",
		toolName, value,
	)
	b, _ := json.Marshal(map[string]string{"reason": msg})
	_, _ = fmt.Fprintln(w, string(b))
}
