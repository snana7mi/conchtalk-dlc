package tools

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"strings"
	"sync"
)

type ExecuteCommandTool struct{}

func (t *ExecuteCommandTool) Name() string { return "execute_command" }
func (t *ExecuteCommandTool) Description() string {
	return "Execute a shell command on the server. Returns stdout, stderr, and exit code."
}
func (t *ExecuteCommandTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"command": map[string]interface{}{
				"type":        "string",
				"description": "The shell command to execute",
			},
		},
		"required": []string{"command"},
	}
}

func (t *ExecuteCommandTool) Execute(ctx context.Context, args map[string]interface{}, stream StreamCallback) ToolResult {
	command, ok := args["command"].(string)
	if !ok || command == "" {
		return ToolResult{Error: "missing or invalid 'command' argument", ExitCode: -1}
	}

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd", "/C", command)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", command)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return ToolResult{Error: fmt.Sprintf("stdout pipe failed: %v", err), ExitCode: -1}
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return ToolResult{Error: fmt.Sprintf("stderr pipe failed: %v", err), ExitCode: -1}
	}

	if err := cmd.Start(); err != nil {
		return ToolResult{Error: fmt.Sprintf("start failed: %v", err), ExitCode: -1}
	}

	var mu sync.Mutex
	var allOutput strings.Builder
	var wg sync.WaitGroup

	readStream := func(r io.Reader, name string) {
		defer wg.Done()
		scanner := bufio.NewScanner(r)
		scanner.Buffer(make([]byte, 64*1024), 1024*1024)
		for scanner.Scan() {
			line := scanner.Text() + "\n"
			mu.Lock()
			allOutput.WriteString(line)
			mu.Unlock()
			stream(name, line)
		}
	}

	wg.Add(2)
	go readStream(stdout, "stdout")
	go readStream(stderr, "stderr")
	wg.Wait()

	exitCode := 0
	if err := cmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return ToolResult{Error: fmt.Sprintf("wait failed: %v", err), ExitCode: -1}
		}
	}

	return ToolResult{
		Output:   allOutput.String(),
		ExitCode: exitCode,
	}
}
