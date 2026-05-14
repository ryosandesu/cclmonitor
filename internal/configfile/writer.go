// Package configfile manipulates cclmonitor.yaml on disk: structured
// insertion via yaml.v3 Node API, timestamped backups, and atomic writes.
package configfile

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Insert appends a single rule to rules.<tool>.<section> in the YAML file at path.
// kind must be "regex" or "glob". Creates the file (and any missing keys) if absent.
// Writes atomically via temp file + rename.
func Insert(path, tool, section, kind, pattern string) error {
	if kind != "regex" && kind != "glob" {
		return fmt.Errorf("invalid kind: %s", kind)
	}
	if section != "allow" && section != "deny" {
		return fmt.Errorf("invalid section: %s", section)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	root, err := loadOrInitRoot(path)
	if err != nil {
		return err
	}

	if err := insertRule(root, tool, section, kind, pattern); err != nil {
		return err
	}

	return atomicWrite(path, root)
}

// Backup copies path to <path>.bak-YYYY-MM-DD-HHMMSS and returns the new path.
// Returns ("", nil) if the source file does not exist (nothing to back up).
func Backup(path string) (string, error) {
	src, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	bak := path + ".bak-" + time.Now().Format("2006-01-02-150405")
	if err := os.WriteFile(bak, src, 0600); err != nil {
		return "", err
	}
	return bak, nil
}

func loadOrInitRoot(path string) (*yaml.Node, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return newEmptyDoc(), nil
	}
	if err != nil {
		return nil, err
	}
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if root.Kind == 0 {
		return newEmptyDoc(), nil
	}
	return &root, nil
}

func newEmptyDoc() *yaml.Node {
	mapping := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	doc := &yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{mapping}}
	return doc
}

func insertRule(root *yaml.Node, tool, section, kind, pattern string) error {
	if root.Kind != yaml.DocumentNode || len(root.Content) == 0 {
		return fmt.Errorf("unexpected document structure")
	}
	top := root.Content[0]
	if top.Kind != yaml.MappingNode {
		return fmt.Errorf("top level must be a mapping")
	}

	rulesNode := findOrCreateMapChild(top, "rules")
	toolNode := findOrCreateMapChild(rulesNode, tool)
	sectionNode := findOrCreateSeqChild(toolNode, section)

	item := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	style := yaml.SingleQuotedStyle
	item.Content = []*yaml.Node{
		{Kind: yaml.ScalarNode, Value: kind, Tag: "!!str"},
		{Kind: yaml.ScalarNode, Value: pattern, Tag: "!!str", Style: style},
	}
	sectionNode.Content = append(sectionNode.Content, item)
	return nil
}

// findOrCreateMapChild finds the value node for a mapping key, creating it
// (as an empty mapping) if absent.
func findOrCreateMapChild(parent *yaml.Node, key string) *yaml.Node {
	for i := 0; i+1 < len(parent.Content); i += 2 {
		if parent.Content[i].Value == key {
			child := parent.Content[i+1]
			if child.Kind == yaml.ScalarNode && child.Value == "" {
				child.Kind = yaml.MappingNode
				child.Tag = "!!map"
			}
			return child
		}
	}
	keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: key, Tag: "!!str"}
	valNode := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	parent.Content = append(parent.Content, keyNode, valNode)
	return valNode
}

// findOrCreateSeqChild finds the value node for a mapping key, creating it
// (as an empty sequence) if absent.
func findOrCreateSeqChild(parent *yaml.Node, key string) *yaml.Node {
	for i := 0; i+1 < len(parent.Content); i += 2 {
		if parent.Content[i].Value == key {
			child := parent.Content[i+1]
			if child.Kind == yaml.ScalarNode && child.Value == "" {
				child.Kind = yaml.SequenceNode
				child.Tag = "!!seq"
			}
			return child
		}
	}
	keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: key, Tag: "!!str"}
	valNode := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
	parent.Content = append(parent.Content, keyNode, valNode)
	return valNode
}

func atomicWrite(path string, root *yaml.Node) error {
	data, err := yaml.Marshal(root)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".cclmonitor.*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer func() {
		_ = os.Remove(tmpPath)
	}()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(0600); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}
