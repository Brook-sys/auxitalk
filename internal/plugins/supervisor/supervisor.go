package supervisor

import (
	"context"
	"fmt"
	"os/exec"
	"sync"
)

type Supervisor struct {
	mu      sync.RWMutex
	plugins map[string]*PluginProcess
}

func NewSupervisor() *Supervisor {
	return &Supervisor{
		plugins: make(map[string]*PluginProcess),
	}
}

func (s *Supervisor) Register(id string, cmd *exec.Cmd) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.plugins[id]; exists {
		return fmt.Errorf("plugin %s already registered", id)
	}

	s.plugins[id] = NewPluginProcess(id, cmd)
	return nil
}

func (s *Supervisor) Start(ctx context.Context, id string) error {
	s.mu.RLock()
	p, ok := s.plugins[id]
	s.mu.RUnlock()

	if !ok {
		return fmt.Errorf("plugin %s not found", id)
	}

	return p.Start(ctx)
}

func (s *Supervisor) Stop(id string) error {
	s.mu.RLock()
	p, ok := s.plugins[id]
	s.mu.RUnlock()

	if !ok {
		return fmt.Errorf("plugin %s not found", id)
	}

	return p.Stop()
}

func (s *Supervisor) IsRunning(id string) bool {
	s.mu.RLock()
	p, ok := s.plugins[id]
	s.mu.RUnlock()

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
