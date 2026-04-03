package tools

import (
	"context"
	"fmt"
)

type GrepSearchTool struct{}

func (t *GrepSearchTool) Name() string { return "grep_search" }
func (t *GrepSearchTool) Description() string {
	return "Search file contents using grep or ripgrep. Returns matching lines with file paths and line numbers."
}
func (t *GrepSearchTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"pattern": map[string]interface{}{"type": "string", "description": "Search pattern (regex)"},
			"path":    map[string]interface{}{"type": "string", "description": "Directory or file to search in (default: current directory)"},
		},
		"required": []string{"pattern"},
	}
}

func (t *GrepSearchTool) Execute(ctx context.Context, args map[string]interface{}, stream StreamCallback) ToolResult {
	pattern, _ := args["pattern"].(string)
	path, _ := args["path"].(string)
	if path == "" {
		path = "."
	}
	cmd := fmt.Sprintf("rg -n %q %s 2>/dev/null || grep -rn %q %s", pattern, path, pattern, path)
	exec := &ExecuteCommandTool{}
	return exec.Execute(ctx, map[string]interface{}{"command": cmd}, stream)
}
