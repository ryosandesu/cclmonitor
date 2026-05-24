package main

import (
	"fmt"
	"io"

	"github.com/ryosandesu/cclmonitor/internal/config"
	"github.com/ryosandesu/cclmonitor/internal/match"
)

func runDryRun(out io.Writer, toolName, value, cwd, globalCfgPath string) int {
	cfg := loadMergedConfig(globalCfgPath, cwd)
	cfg = config.ExpandCwd(cfg, cwd)

	rules := cfg.Rules[toolName]

	var verdict match.Verdict
	var err error
	if toolName == "Bash" {
		tokens := match.SplitBashCommands(value)
		verdict, err = evaluateTokens(rules, tokens)
	} else {
		verdict, err = match.Evaluate(rules, value)
	}
	if err != nil {
		fmt.Fprintf(out, "error: %v\n", err)
		return 1
	}

	if verdict == match.Unknown && cfg.DefaultVerdict == "deny" {
		verdict = match.Deny
	}

	fmt.Fprintf(out, "tool:    %s\n", toolName)
	fmt.Fprintf(out, "value:   %s\n", value)
	fmt.Fprintf(out, "verdict: %s\n", verdictString(verdict))
	return 0
}
