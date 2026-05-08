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
	verdict, err := match.Evaluate(rules, value)
	if err != nil {
		fmt.Fprintf(out, "error: %v\n", err)
		return 1
	}

	fmt.Fprintf(out, "tool:    %s\n", toolName)
	fmt.Fprintf(out, "value:   %s\n", value)
	fmt.Fprintf(out, "verdict: %s\n", verdictString(verdict))
	return 0
}
