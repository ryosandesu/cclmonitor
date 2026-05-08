package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	path := settingsPath()
	if err := injectHook(path); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	fmt.Println("cclmonitor hook registered in", path)
}

func settingsPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".claude/settings.json"
	}
	return filepath.Join(home, ".claude", "settings.json")
}
