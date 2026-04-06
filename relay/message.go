package relay

import "encoding/json"

// IncomingMessage represents messages received from the relay server.
type IncomingMessage struct {
	Type      string          `json:"type"`
	ID        string          `json:"id,omitempty"`
	Tool      string          `json:"tool,omitempty"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
	Client    string          `json:"client,omitempty"`
	// ACP session fields
	AgentType string `json:"agent_type,omitempty"`
	Command   string `json:"command,omitempty"`
	Cwd       string `json:"cwd,omitempty"`
	Data      string `json:"data,omitempty"`
	SessionID string `json:"session_id,omitempty"`
}

// OutgoingMessage represents messages sent to the relay server.
type OutgoingMessage struct {
	Type   string `json:"type"`
	ID     string `json:"id,omitempty"`
	Stream string `json:"stream,omitempty"`
	Data   string `json:"data,omitempty"`

	// For tool_done
	ExitCode *int   `json:"exit_code,omitempty"`
	Output   string `json:"output,omitempty"`

	// For tool_error
	Error string `json:"error,omitempty"`

	// For capabilities
	Tools  []ToolDefinition  `json:"tools,omitempty"`
	Skills []SkillDefinition `json:"skills,omitempty"`
	Agents []AgentDefinition `json:"agents,omitempty"`

	// ACP session fields
	SessionID string `json:"session_id,omitempty"`
	AgentType string `json:"agent_type,omitempty"`
}

// ToolDefinition describes a tool exposed to the relay server.
type ToolDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// SkillDefinition describes a skill exposed to the relay server.
type SkillDefinition struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Description string `json:"description"`
	Content     string `json:"content"`
}

// AgentDefinition describes an ACP-compatible coding agent available on the server.
type AgentDefinition struct {
	Type    string `json:"type"`
	Name    string `json:"name"`
	Path    string `json:"path"`
	Version string `json:"version,omitempty"`
}
