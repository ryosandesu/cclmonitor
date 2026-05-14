package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

var version = "dev"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Println("cclmonitor", version)
		os.Exit(0)
	}
	if len(os.Args) > 1 && os.Args[1] == "test" {
		os.Exit(runTestCmd(os.Args[2:]))
	}
	if len(os.Args) > 1 && os.Args[1] == "suggest" {
		os.Exit(runSuggestCmd(os.Args[2:]))
	}
	if len(os.Args) > 1 && os.Args[1] == "post" {
		os.Exit(runPost(os.Stdin, globalCfgPath()))
	}
	os.Exit(run(os.Stdin, os.Stdout, globalCfgPath()))
}

func runTestCmd(args []string) int {
	fs := flag.NewFlagSet("test", flag.ExitOnError)
	tool := fs.String("tool", "Bash", "tool name (Bash/Edit/Write/Read)")
	cwd := fs.String("cwd", mustGetwd(), "working directory for <cwd> expansion")
	_ = fs.Parse(args)

	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "usage: cclmonitor test [--tool TOOL] [--cwd DIR] <value>")
		return 1
	}
	value := fs.Arg(0)
	return runDryRun(os.Stdout, *tool, value, *cwd, globalCfgPath())
}

func mustGetwd() string {
	cwd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return cwd
}

func globalCfgPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".claude", "cclmonitor.yaml")
}
