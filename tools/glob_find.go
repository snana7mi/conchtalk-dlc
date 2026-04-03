package tools

import (
	"context"
	"fmt"
	"strings"
)

type GlobFindTool struct{}

func (t *GlobFindTool) Name() string { return "glob_find" }
func (t *GlobFindTool) Description() string {
	return "Find files matching a glob pattern."
}
func (t *GlobFindTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"pattern": map[string]interface{}{"type": "string", "description": "Glob pattern (e.g. '*.go', '**/*.go')"},
			"path":    map[string]interface{}{"type": "string", "description": "Base directory (default: current directory)"},
		},
		"required": []string{"pattern"},
	}
}

func (t *GlobFindTool) Execute(ctx context.Context, args map[string]interface{}, stream StreamCallback) ToolResult {
	pattern, _ := args["pattern"].(string)
	path, _ := args["path"].(string)
	if path == "" {
		path = "."
	}
	// Strip leading **/ for find compatibility, use -name for the file part
	cleanPattern := strings.TrimPrefix(pattern, "**/")
	var cmd string
	if strings.Contains(cleanPattern, "/") {
		cmd = fmt.Sprintf("find %q -path %q -type f 2>/dev/null | head -200", path, "*"+cleanPattern)
	} else {
		cmd = fmt.Sprintf("find %q -name %q -type f 2>/dev/null | head -200", path, cleanPattern)
	}
	exec := &ExecuteCommandTool{}
	return exec.Execute(ctx, map[string]interface{}{"command": cmd}, stream)
}
