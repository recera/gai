// Package tools provides an adapter to convert between core and tools package types.
// This file helps avoid circular dependencies between core and tools packages.

package tools

import (
	"context"
	"encoding/json"
	
	"github.com/recera/gai/core"
)

// CoreToolAdapter wraps a tools.Handle to implement core.ToolHandle.
// This allows tools defined in the tools package to be used with the core runner.
type CoreToolAdapter struct {
	tool Handle
}

// NewCoreAdapter creates an adapter that wraps a tools.Handle for use with core.
func NewCoreAdapter(tool Handle) core.ToolHandle {
	return &CoreToolAdapter{tool: tool}
}

// Name returns the tool's name.
func (a *CoreToolAdapter) Name() string {
	return a.tool.Name()
}

// Description returns the tool's description.
func (a *CoreToolAdapter) Description() string {
	return a.tool.Description()
}

// InSchemaJSON returns the input schema.
func (a *CoreToolAdapter) InSchemaJSON() []byte {
	return a.tool.InSchemaJSON()
}

// OutSchemaJSON returns the output schema.
func (a *CoreToolAdapter) OutSchemaJSON() []byte {
	return a.tool.OutSchemaJSON()
}

// Exec executes the tool, converting the meta interface{} to our Meta type.
func (a *CoreToolAdapter) Exec(ctx context.Context, raw json.RawMessage, metaInterface interface{}) (any, error) {
	// Convert the interface{} meta to our Meta struct
	meta := Meta{}
	
	// Try to extract fields from the meta interface
	if m, ok := metaInterface.(map[string]interface{}); ok {
		if callID, ok := m["call_id"].(string); ok {
			meta.CallID = callID
		}
		if messages, ok := m["messages"].([]core.Message); ok {
			meta.Messages = messages
		}
		if stepNumber, ok := m["step_number"].(int); ok {
			meta.StepNumber = stepNumber
		}
		if provider, ok := m["provider"].(string); ok {
			meta.Provider = provider
		}
		if metadata, ok := m["metadata"].(map[string]any); ok {
			meta.Metadata = metadata
		}
	}
	
	// Execute the underlying tool
	return a.tool.Exec(ctx, raw, meta)
}

// ToHandles converts a slice of core.ToolHandle to tools.Handle.
// This is useful when you need to work with tools in the tools package context.
func ToHandles(coreTools []core.ToolHandle) []Handle {
	handles := make([]Handle, 0, len(coreTools))
	for _, ct := range coreTools {
		// Check if it's already an adapter
		if adapter, ok := ct.(*CoreToolAdapter); ok {
			handles = append(handles, adapter.tool)
		}
		// Otherwise, create a generic handle wrapper
		// This case shouldn't normally happen if tools are created properly
	}
	return handles
}

// ToCoreHandles converts a slice of tools.Handle to core.ToolHandle.
// This is the recommended way to pass tools to the core runner.
func ToCoreHandles(tools []Handle) []core.ToolHandle {
	coreTools := make([]core.ToolHandle, len(tools))
	for i, tool := range tools {
		coreTools[i] = NewCoreAdapter(tool)
	}
	return coreTools
}