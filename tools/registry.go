package tools

import (
	"context"
	"fmt"

	"github.com/snana7mi/conchtalk-dlc/relay"
)

type ToolResult struct {
	Output   string
	ExitCode int
	Error    string
}

type StreamCallback func(stream string, data string)

type Tool interface {
	Name() string
	Description() string
	Parameters() map[string]interface{}
	Execute(ctx context.Context, args map[string]interface{}, stream StreamCallback) ToolResult
}

type Registry struct {
	tools map[string]Tool
}

func NewRegistry() *Registry {
	r := &Registry{tools: make(map[string]Tool)}
	r.Register(&ExecuteCommandTool{})
	r.Register(&ReadFileTool{})
	r.Register(&WriteFileTool{})
	r.Register(&ListDirectoryTool{})
	r.Register(&GrepSearchTool{})
	r.Register(&GlobFindTool{})
	r.Register(&SystemInfoTool{})
	return r
}

func (r *Registry) Register(t Tool) {
	r.tools[t.Name()] = t
}

func (r *Registry) Get(name string) (Tool, error) {
	t, ok := r.tools[name]
	if !ok {
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
	return t, nil
}

func (r *Registry) Definitions() []relay.ToolDefinition {
	defs := make([]relay.ToolDefinition, 0, len(r.tools))
	for _, t := range r.tools {
		defs = append(defs, relay.ToolDefinition{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters:  t.Parameters(),
		})
	}
	return defs
}
