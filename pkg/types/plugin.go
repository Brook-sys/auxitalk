package types

import (
	"errors"
	"strings"
)

type PluginKind string

const (
	PluginKindInput   PluginKind = "input"
	PluginKindOutput  PluginKind = "output"
	PluginKindAI      PluginKind = "ai"
	PluginKindMemory  PluginKind = "memory"
	PluginKindUI      PluginKind = "ui"
	PluginKindPolicy  PluginKind = "policy"
	PluginKindTool    PluginKind = "tool"
	PluginKindProfile PluginKind = "profile"
)

type Capability struct {
	Name         string         `json:"name"`
	InputSchema  map[string]any `json:"inputSchema,omitempty"`
	OutputSchema map[string]any `json:"outputSchema,omitempty"`
}

func (c Capability) Validate() error {
	if strings.TrimSpace(c.Name) == "" {
		return errors.New("capability name is required")
	}
	return nil
}

type PluginManifest struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	Version      string       `json:"version"`
	Runtime      string       `json:"runtime"`
	Entry        string       `json:"entry"`
	Kind         PluginKind   `json:"kind"`
	Permissions  []string     `json:"permissions,omitempty"`
	Capabilities []Capability `json:"capabilities,omitempty"`
}

func (m PluginManifest) Validate() error {
	if strings.TrimSpace(m.ID) == "" {
		return errors.New("plugin id is required")
	}
	if strings.TrimSpace(m.Name) == "" {
		return errors.New("plugin name is required")
	}
	if strings.TrimSpace(m.Version) == "" {
		return errors.New("plugin version is required")
	}
	if strings.TrimSpace(m.Runtime) == "" {
		return errors.New("plugin runtime is required")
	}
	switch m.Kind {
	case PluginKindInput, PluginKindOutput, PluginKindAI, PluginKindMemory, PluginKindUI, PluginKindPolicy, PluginKindTool, PluginKindProfile:
	default:
		return errors.New("plugin kind is invalid")
	}
	for _, capability := range m.Capabilities {
		if err := capability.Validate(); err != nil {
			return err
		}
	}
	for _, permission := range m.Permissions {
		if strings.TrimSpace(permission) == "" {
			return errors.New("plugin permission cannot be empty")
		}
	}
	return nil
}
