package acp

import (
	"os/exec"
	"strings"

	"github.com/snana7mi/conchtalk-dlc/relay"
)

// knownAgents lists ACP-compatible coding agents to detect.
var knownAgents = []struct {
	Type    string
	Name    string
	Binary  string
	AcpFlag string
}{
	{"claude", "Claude Code", "claude", "--acp"},
	{"codex", "Codex", "codex", ""},
	{"gemini", "Gemini CLI", "gemini", "--acp"},
	{"kimi", "Kimi CLI", "kimi", "--acp"},
	{"opencode", "OpenCode", "opencode", "--acp"},
	{"openclaw", "OpenClaw", "openclaw", "--acp"},
	{"qwen", "Qwen Code", "qwen", "--acp"},
}

// DetectAgents scans PATH for available coding agents.
func DetectAgents() []relay.AgentDefinition {
	var agents []relay.AgentDefinition
	for _, a := range knownAgents {
		path, err := exec.LookPath(a.Binary)
		if err != nil {
			continue
		}
		version := ""
		out, err := exec.Command(path, "--version").Output()
		if err == nil {
			version = strings.TrimSpace(string(out))
			if idx := strings.IndexByte(version, '\n'); idx >= 0 {
				version = version[:idx]
			}
		}
		agents = append(agents, relay.AgentDefinition{
			Type:    a.Type,
			Name:    a.Name,
			Path:    path,
			Version: version,
		})
	}
	return agents
}
