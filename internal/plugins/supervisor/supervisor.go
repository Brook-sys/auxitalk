package supervisor

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
)

type Supervisor struct {
	mu      sync.RWMutex
	plugins map[string]*PluginProcess
	opts    ProcessOptions
}

func NewSupervisor(opts ...ProcessOptions) *Supervisor {
	var options ProcessOptions
	if len(opts) > 0 {
		options = opts[0]
	}
	return &Supervisor{
		plugins: make(map[string]*PluginProcess),
		opts:    options,
	}
}

func (s *Supervisor) Register(spec ProcessSpec) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.plugins[spec.ID]; exists {
		return fmt.Errorf("plugin %s already registered", spec.ID)
	}

	s.plugins[spec.ID] = NewPluginProcess(spec, s.opts)
	return nil
}

func (s *Supervisor) Start(ctx context.Context, id string) error {
	p, ok := s.get(id)
	if !ok {
		return fmt.Errorf("plugin %s not found", id)
	}
	return p.Start(ctx)
}

func (s *Supervisor) Stop(id string) error {
	p, ok := s.get(id)
	if !ok {
		return fmt.Errorf("plugin %s not found", id)
	}
	return p.Stop()
}

func (s *Supervisor) Call(ctx context.Context, id string, method string, params any) (json.RawMessage, error) {
	p, ok := s.get(id)
	if !ok {
		return nil, fmt.Errorf("plugin %s not found", id)
	}
	return p.Call(ctx, method, params)
}

func (s *Supervisor) HealthCheck(ctx context.Context, id string) error {
	_, err := s.Call(ctx, id, "plugin.health", nil)
	return err
}

func (s *Supervisor) IsRunning(id string) bool {
	p, ok := s.get(id)
	if !ok {
		return false
	}
	return p.IsRunning()
}

func (s *Supervisor) List() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ids := make([]string, 0, len(s.plugins))
	for id := range s.plugins {
		ids = append(ids, id)
	}
	return ids
}

func (s *Supervisor) Status(id string) (ProcessStatus, bool) {
	p, ok := s.get(id)
	if !ok {
		return ProcessStatus{}, false
	}
	return p.Status(), true
}

func (s *Supervisor) ListStatus() []ProcessStatus {
	s.mu.RLock()
	plugins := make([]*PluginProcess, 0, len(s.plugins))
	for _, plugin := range s.plugins {
		plugins = append(plugins, plugin)
	}
	s.mu.RUnlock()

	statuses := make([]ProcessStatus, 0, len(plugins))
	for _, plugin := range plugins {
		statuses = append(statuses, plugin.Status())
	}
	return statuses
}

func (s *Supervisor) get(id string) (*PluginProcess, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.plugins[id]
	return p, ok
}
