package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
)

func injectHook(path string) error {
	var raw map[string]interface{}

	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if err == nil {
		if werr := os.WriteFile(path+".bak", data, 0600); werr != nil {
			return werr
		}
		if err := json.Unmarshal(data, &raw); err != nil {
			return err
		}
	} else {
		raw = map[string]interface{}{}
	}

	if alreadyInjected(raw) {
		return nil
	}

	addHookEntry(raw)

	out, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(out, '\n'), 0600)
}

func cclmonitorPath() string {
	name := binaryName("cclmonitor", runtime.GOOS)
	exe, err := os.Executable()
	if err != nil {
		return name
	}
	return filepath.Join(filepath.Dir(exe), name)
}

func binaryName(name, goos string) string {
	if goos == "windows" {
		return name + ".exe"
	}
	return name
}

func alreadyInjected(raw map[string]interface{}) bool {
	hooks, _ := raw["hooks"].(map[string]interface{})
	if hooks == nil {
		return false
	}
	target := cclmonitorPath()
	return hasCommand(hooks["PreToolUse"], target) &&
		hasCommand(hooks["PostToolUse"], target+" post")
}

func hasCommand(hookSection interface{}, cmd string) bool {
	entries, _ := hookSection.([]interface{})
	for _, entry := range entries {
		entryMap, _ := entry.(map[string]interface{})
		if entryMap == nil {
			continue
		}
		hookList, _ := entryMap["hooks"].([]interface{})
		for _, h := range hookList {
			hMap, _ := h.(map[string]interface{})
			if hMap["command"] == cmd {
				return true
			}
		}
	}
	return false
}

func addHookEntry(raw map[string]interface{}) {
	hooks, _ := raw["hooks"].(map[string]interface{})
	if hooks == nil {
		hooks = map[string]interface{}{}
		raw["hooks"] = hooks
	}

	target := cclmonitorPath()

	if !hasCommand(hooks["PreToolUse"], target) {
		preToolUse, _ := hooks["PreToolUse"].([]interface{})
		hooks["PreToolUse"] = append(preToolUse, map[string]interface{}{
			"matcher": "",
			"hooks": []interface{}{
				map[string]interface{}{
					"type":    "command",
					"command": target,
				},
			},
		})
	}

	if !hasCommand(hooks["PostToolUse"], target+" post") {
		postToolUse, _ := hooks["PostToolUse"].([]interface{})
		hooks["PostToolUse"] = append(postToolUse, map[string]interface{}{
			"matcher": "",
			"hooks": []interface{}{
				map[string]interface{}{
					"type":    "command",
					"command": target + " post",
				},
			},
		})
	}
}
