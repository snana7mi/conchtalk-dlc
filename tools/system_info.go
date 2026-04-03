package tools

import (
	"context"
	"fmt"
	"runtime"
	"strings"
)

type SystemInfoTool struct{}

func (t *SystemInfoTool) Name() string { return "system_info" }
func (t *SystemInfoTool) Description() string {
	return "Get system information including OS, architecture, CPU count, memory, disk, and uptime."
}
func (t *SystemInfoTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
	}
}

func (t *SystemInfoTool) Execute(ctx context.Context, args map[string]interface{}, stream StreamCallback) ToolResult {
	var output strings.Builder

	// Basic info from Go runtime
	output.WriteString(fmt.Sprintf("OS:      %s\n", runtime.GOOS))
	output.WriteString(fmt.Sprintf("Arch:    %s\n", runtime.GOARCH))
	output.WriteString(fmt.Sprintf("CPUs:    %d\n", runtime.NumCPU()))
	output.WriteString(fmt.Sprintf("Go:      %s\n", runtime.Version()))

	// Additional info from shell commands
	exec := &ExecuteCommandTool{}
	noopStream := func(string, string) {}

	uname := exec.Execute(ctx, map[string]interface{}{"command": "uname -a 2>/dev/null"}, noopStream)
	if uname.ExitCode == 0 && uname.Output != "" {
		output.WriteString(fmt.Sprintf("Uname:   %s", uname.Output))
	}

	// Memory info (platform-dependent)
	if runtime.GOOS == "darwin" {
		mem := exec.Execute(ctx, map[string]interface{}{"command": "sysctl -n hw.memsize 2>/dev/null"}, noopStream)
		if mem.ExitCode == 0 && mem.Output != "" {
			output.WriteString(fmt.Sprintf("Memory:  %s bytes\n", strings.TrimSpace(mem.Output)))
		}
	} else {
		mem := exec.Execute(ctx, map[string]interface{}{"command": "free -h 2>/dev/null | head -2"}, noopStream)
		if mem.ExitCode == 0 && mem.Output != "" {
			output.WriteString(fmt.Sprintf("Memory:\n%s", mem.Output))
		}
	}

	disk := exec.Execute(ctx, map[string]interface{}{"command": "df -h / 2>/dev/null"}, noopStream)
	if disk.ExitCode == 0 && disk.Output != "" {
		output.WriteString(fmt.Sprintf("Disk:\n%s", disk.Output))
	}

	uptime := exec.Execute(ctx, map[string]interface{}{"command": "uptime 2>/dev/null"}, noopStream)
	if uptime.ExitCode == 0 && uptime.Output != "" {
		output.WriteString(fmt.Sprintf("Uptime:  %s", uptime.Output))
	}

	result := output.String()
	stream("stdout", result)
	return ToolResult{Output: result, ExitCode: 0}
}
