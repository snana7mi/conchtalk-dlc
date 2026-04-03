package tools

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
)

type ReadFileTool struct{}

func (t *ReadFileTool) Name() string { return "read_file" }
func (t *ReadFileTool) Description() string {
	return "Read the contents of a file. Supports optional line range with start_line and end_line."
}
func (t *ReadFileTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Absolute or relative path to the file",
			},
			"start_line": map[string]interface{}{
				"type":        "number",
				"description": "Starting line number (1-based, inclusive)",
			},
			"end_line": map[string]interface{}{
				"type":        "number",
				"description": "Ending line number (1-based, inclusive)",
			},
		},
		"required": []string{"path"},
	}
}

func toInt(v interface{}) (int, bool) {
	switch n := v.(type) {
	case float64:
		return int(n), true
	case int:
		return n, true
	case int64:
		return int(n), true
	default:
		return 0, false
	}
}

func (t *ReadFileTool) Execute(ctx context.Context, args map[string]interface{}, stream StreamCallback) ToolResult {
	path, ok := args["path"].(string)
	if !ok || path == "" {
		return ToolResult{Error: "missing or invalid 'path' argument", ExitCode: -1}
	}

	startLine := 0
	endLine := 0
	if v, ok := args["start_line"]; ok {
		if n, ok := toInt(v); ok {
			startLine = n
		}
	}
	if v, ok := args["end_line"]; ok {
		if n, ok := toInt(v); ok {
			endLine = n
		}
	}

	file, err := os.Open(path)
	if err != nil {
		return ToolResult{Error: fmt.Sprintf("failed to open file: %v", err), ExitCode: -1}
	}
	defer file.Close()

	var output strings.Builder
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		if startLine > 0 && lineNum < startLine {
			continue
		}
		if endLine > 0 && lineNum > endLine {
			break
		}
		line := fmt.Sprintf("%d\t%s\n", lineNum, scanner.Text())
		output.WriteString(line)
		stream("stdout", line)
	}

	if err := scanner.Err(); err != nil {
		return ToolResult{Error: fmt.Sprintf("error reading file: %v", err), ExitCode: -1}
	}

	return ToolResult{Output: output.String(), ExitCode: 0}
}
