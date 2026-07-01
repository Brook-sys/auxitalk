package supervisor

import (
	"context"
	"errors"
	"os/exec"
	"sync"
)

var (
	ErrAlreadyRunning = errors.New("plugin already running")
	ErrNotRunning     = errors.New("plugin is not running")
)

type PluginProcess struct {
	ID      string
	Cmd     *exec.Cmd
	Running bool
	mu      sync.Mutex
}

func NewPluginProcess(id string, cmd *exec.Cmd) *PluginProcess {
	return &PluginProcess{
		ID:  id,
		Cmd: cmd,
	}
}

func (p *PluginProcess) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.Running {
		return ErrAlreadyRunning
	}

	if err := p.Cmd.Start(); err != nil {
		return err
	}

	p.Running = true
	return nil
}

func (p *PluginProcess) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.Running {
		return ErrNotRunning
	}

	if p.Cmd.Process != nil {
		_ = p.Cmd.Process.Kill()
	}

	p.Running = false
	return nil
}

func (p *PluginProcess) IsRunning() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.Running
}
