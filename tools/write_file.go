package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

type WriteFileTool struct{}

func (t *WriteFileTool) Name() string { return "write_file" }
func (t *WriteFileTool) Description() string {
	return "Write content to a file. Creates parent directories if needed. Supports append mode."
}
func (t *WriteFileTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Absolute or relative path to the file",
			},
			"content": map[string]interface{}{
				"type":        "string",
				"description": "Content to write to the file",
			},
			"append": map[string]interface{}{
				"type":        "boolean",
				"description": "If true, append to the file instead of overwriting (default: false)",
			},
		},
		"required": []string{"path", "content"},
	}
}

func (t *WriteFileTool) Execute(ctx context.Context, args map[string]interface{}, stream StreamCallback) ToolResult {
	path, ok := args["path"].(string)
	if !ok || path == "" {
		return ToolResult{Error: "missing or invalid 'path' argument", ExitCode: -1}
	}

	content, ok := args["content"].(string)
	if !ok {
		return ToolResult{Error: "missing or invalid 'content' argument", ExitCode: -1}
	}

	appendMode, _ := args["append"].(bool)

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return ToolResult{Error: fmt.Sprintf("failed to create directories: %v", err), ExitCode: -1}
	}

	flag := os.O_WRONLY | os.O_CREATE
	if appendMode {
		flag |= os.O_APPEND
	} else {
		flag |= os.O_TRUNC
	}

	file, err := os.OpenFile(path, flag, 0644)
	if err != nil {
		return ToolResult{Error: fmt.Sprintf("failed to open file: %v", err), ExitCode: -1}
	}
	defer file.Close()

	n, err := file.WriteString(content)
	if err != nil {
		return ToolResult{Error: fmt.Sprintf("failed to write file: %v", err), ExitCode: -1}
	}

	msg := fmt.Sprintf("Wrote %d bytes to %s", n, path)
	stream("stdout", msg+"\n")
	return ToolResult{Output: msg, ExitCode: 0}
}
