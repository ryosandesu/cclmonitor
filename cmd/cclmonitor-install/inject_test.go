package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInjectHook_CreatesFileIfMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")

	if err := injectHook(path); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal("settings.json should be created")
	}
	if !strings.Contains(string(data), "cclmonitor") {
		t.Errorf("settings.json should contain cclmonitor, got: %s", data)
	}
	if _, err := os.Stat(path + ".bak"); err == nil {
		t.Error("backup should not be created when file did not exist")
	}
}

func TestInjectHook_BacksUpExistingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	original := `{"model":"claude-opus-4"}`
	if err := os.WriteFile(path, []byte(original), 0600); err != nil {
		t.Fatal(err)
	}

	if err := injectHook(path); err != nil {
		t.Fatal(err)
	}

	bak, err := os.ReadFile(path + ".bak")
	if err != nil {
		t.Fatal("backup should exist")
	}
	if string(bak) != original {
		t.Errorf("backup content = %q, want %q", bak, original)
	}
}

func TestInjectHook_PreservesExistingFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(path, []byte(`{"model":"claude-opus-4"}`), 0600); err != nil {
		t.Fatal(err)
	}

	if err := injectHook(path); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(path)
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	if raw["model"] != "claude-opus-4" {
		t.Errorf("model field should be preserved, got: %v", raw["model"])
	}
}

func TestInjectHook_AddsPreToolUseHook(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")

	if err := injectHook(path); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(path)
	var raw map[string]interface{}
	json.Unmarshal(data, &raw)

	hooks, _ := raw["hooks"].(map[string]interface{})
	preToolUse, _ := hooks["PreToolUse"].([]interface{})
	if len(preToolUse) == 0 {
		t.Fatal("PreToolUse should have at least one entry")
	}
	entry, _ := preToolUse[0].(map[string]interface{})
	hookList, _ := entry["hooks"].([]interface{})
	if len(hookList) == 0 {
		t.Fatal("hook entry should have at least one hook")
	}
	h, _ := hookList[0].(map[string]interface{})
	cmd, _ := h["command"].(string)
	if !strings.HasSuffix(cmd, "cclmonitor") {
		t.Errorf("command = %v, want path ending with cclmonitor", cmd)
	}
	if h["type"] != "command" {
		t.Errorf("type = %v, want command", h["type"])
	}
}

func TestInjectHook_Idempotent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")

	if err := injectHook(path); err != nil {
		t.Fatal(err)
	}
	if err := injectHook(path); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(path)
	var raw map[string]interface{}
	json.Unmarshal(data, &raw)
	hooks, _ := raw["hooks"].(map[string]interface{})
	preToolUse, _ := hooks["PreToolUse"].([]interface{})
	count := 0
	for _, entry := range preToolUse {
		entryMap, _ := entry.(map[string]interface{})
		for _, h := range entryMap["hooks"].([]interface{}) {
			hMap, _ := h.(map[string]interface{})
			if cmd, _ := hMap["command"].(string); strings.HasSuffix(cmd, "cclmonitor") {
				count++
			}
		}
	}
	if count != 1 {
		t.Errorf("cclmonitor should appear once, got %d times\n%s", count, data)
	}
}

func TestInjectHook_AppendsToExistingPreToolUse(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	existing := `{
  "hooks": {
    "PreToolUse": [
      {"matcher":"Bash","hooks":[{"type":"command","command":"other-tool"}]}
    ]
  }
}`
	if err := os.WriteFile(path, []byte(existing), 0600); err != nil {
		t.Fatal(err)
	}

	if err := injectHook(path); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(path)
	var raw map[string]interface{}
	json.Unmarshal(data, &raw)

	hooks, _ := raw["hooks"].(map[string]interface{})
	preToolUse, _ := hooks["PreToolUse"].([]interface{})
	if len(preToolUse) != 2 {
		t.Errorf("PreToolUse should have 2 entries, got %d", len(preToolUse))
	}
}
