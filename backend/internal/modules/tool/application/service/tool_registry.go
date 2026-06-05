package service

import (
	"fmt"

	"github.com/cloudwego/eino/schema"
)

// ToolDef is a registered tool — its Eino metadata plus a stateless executor.
type ToolDef struct {
	Info    *schema.ToolInfo
	Execute func(argsJSON string) (string, error)
}

// Registry maps tool names to their definitions.
type Registry struct {
	tools map[string]*ToolDef
}

func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]*ToolDef)}
}

func (r *Registry) Register(def *ToolDef) {
	r.tools[def.Info.Name] = def
}

// Get returns a tool by name, or nil if not found.
func (r *Registry) Get(name string) *ToolDef {
	return r.tools[name]
}

// InfoFor returns the *schema.ToolInfo for a list of tool names (skips unknowns).
func (r *Registry) InfoFor(names []string) []*schema.ToolInfo {
	var out []*schema.ToolInfo
	for _, n := range names {
		if def, ok := r.tools[n]; ok {
			out = append(out, def.Info)
		}
	}
	return out
}

// Execute runs the named tool with the provided JSON arguments string.
func (r *Registry) Execute(name, argsJSON string) (string, error) {
	def, ok := r.tools[name]
	if !ok {
		return "", fmt.Errorf("unknown tool: %s", name)
	}
	return def.Execute(argsJSON)
}

// All returns all registered tool infos — used for listing available tools.
func (r *Registry) All() []*schema.ToolInfo {
	out := make([]*schema.ToolInfo, 0, len(r.tools))
	for _, def := range r.tools {
		out = append(out, def.Info)
	}
	return out
}
