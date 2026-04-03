package tools

import (
	"context"
	"fmt"
	"os"
	"strings"
)

type ListDirectoryTool struct{}

func (t *ListDirectoryTool) Name() string { return "list_directory" }
func (t *ListDirectoryTool) Description() string {
	return "List the contents of a directory with file mode, size, and name."
}
func (t *ListDirectoryTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Directory path to list (default: current directory)",
			},
		},
		"required": []string{"path"},
	}
}

func (t *ListDirectoryTool) Execute(ctx context.Context, args map[string]interface{}, stream StreamCallback) ToolResult {
	path, _ := args["path"].(string)
	if path == "" {
		path = "."
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return ToolResult{Error: fmt.Sprintf("failed to read directory: %v", err), ExitCode: -1}
	}

	var output strings.Builder
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		line := fmt.Sprintf("%s\t%d\t%s\n", info.Mode(), info.Size(), entry.Name())
		output.WriteString(line)
		stream("stdout", line)
	}

	return ToolResult{Output: output.String(), ExitCode: 0}
}
