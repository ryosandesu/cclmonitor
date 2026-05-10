package hookio

import (
	"encoding/json"
	"io"
)

type ToolResponse struct {
	Interrupted bool `json:"interrupted"`
}

type HookPayload struct {
	ToolName     string          `json:"tool_name"`
	ToolInput    json.RawMessage `json:"tool_input"`
	Cwd          string          `json:"cwd"`
	SessionID    string          `json:"session_id"`
	ToolUseID    string          `json:"tool_use_id"`
	ToolResponse ToolResponse    `json:"tool_response"`
}

type BashInput struct {
	Command string `json:"command"`
}

type EditInput struct {
	FilePath  string `json:"file_path"`
	OldString string `json:"old_string"`
	NewString string `json:"new_string"`
}

type WriteInput struct {
	FilePath string `json:"file_path"`
	Content  string `json:"content"`
}

type ReadInput struct {
	FilePath string `json:"file_path"`
}

func Parse(r io.Reader) (*HookPayload, error) {
	var p HookPayload
	if err := json.NewDecoder(r).Decode(&p); err != nil {
		return nil, err
	}
	return &p, nil
}

// MatchValue extracts the value to match against rules.
// Bash → command string, Edit/Write/Read → file_path.
func MatchValue(p *HookPayload) (string, error) {
	switch p.ToolName {
	case "Bash":
		var in BashInput
		if err := json.Unmarshal(p.ToolInput, &in); err != nil {
			return "", err
		}
		return in.Command, nil
	case "Edit":
		var in EditInput
		if err := json.Unmarshal(p.ToolInput, &in); err != nil {
			return "", err
		}
		return in.FilePath, nil
	case "Write":
		var in WriteInput
		if err := json.Unmarshal(p.ToolInput, &in); err != nil {
			return "", err
		}
		return in.FilePath, nil
	case "Read":
		var in ReadInput
		if err := json.Unmarshal(p.ToolInput, &in); err != nil {
			return "", err
		}
		return in.FilePath, nil
	default:
		return "", nil
	}
}
