package plugins

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/Brook-sys/auxitalk/pkg/types"
)

var (
	ErrSupervisorAlreadyRunning = errors.New("plugin supervisor already running")
	ErrSupervisorNotRunning     = errors.New("plugin supervisor is not running")
)

type SupervisorOptions struct {
	HealthInterval time.Duration
	HealthTimeout  time.Duration
	RestartBackoff time.Duration
	MaxRestarts    int
}

type Supervisor struct {
	plugin   Plugin
	opts     SupervisorOptions
	cmd      *exec.Cmd
	stdin    io.WriteCloser
	stdout   io.ReadCloser
	stderr   io.ReadCloser
	mu       sync.RWMutex
	running  bool
	restarts int
	ctx      context.Context
	cancel   context.CancelFunc
}

func NewSupervisor(plugin Plugin, opts SupervisorOptions) *Supervisor {
	if opts.HealthInterval <= 0 {
		opts.HealthInterval = 30 * time.Second
	}
	if opts.HealthTimeout <= 0 {
		opts.HealthTimeout = 2 * time.Second
	}
	if opts.RestartBackoff <= 0 {
		opts.RestartBackoff = 1 * time.Second
	}
	if opts.MaxRestarts <= 0 {
		opts.MaxRestarts = 3
	}

	return &Supervisor{
		plugin: plugin,
		opts:   opts,
	}
}

func (s *Supervisor) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return ErrSupervisorAlreadyRunning
	}

	if err := s.plugin.Manifest.Validate(); err != nil {
		return fmt.Errorf("invalid manifest: %w", err)
	}

	entryPath := s.plugin.Manifest.Entry
	if !filepath.IsAbs(entryPath) {
		entryPath = filepath.Join(s.plugin.RootDir, entryPath)
	}

	cmd := exec.CommandContext(ctx, s.plugin.Manifest.Runtime, entryPath)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start plugin: %w", err)
	}

	s.cmd = cmd
	s.stdin, _ = stdin.(*os.File)
	s.stdout, _ = stdout.(*os.File)
	s.stderr, _ = stderr.(*os.File)
	s.running = true
	s.restarts = 0
	s.ctx, s.cancel = context.WithCancel(ctx)

	go s.monitor()

	return nil
}

func (s *Supervisor) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return ErrSupervisorNotRunning
	}

	s.cancel()
	if s.cmd != nil && s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
	}
	s.running = false
	return nil
}

func (s *Supervisor) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

func (s *Supervisor) Plugin() Plugin {
	return s.plugin
}

func (s *Supervisor) Manifest() types.PluginManifest {
	return s.plugin.Manifest
}

func (s *Supervisor) monitor() {
	s.cmd.Wait()

	s.mu.Lock()
	defer s.mu.Unlock()

	s.running = false
	if s.restarts < s.opts.MaxRestarts {
		s.restarts++
		time.Sleep(s.opts.RestartBackoff)
		// restart logic would go here in full implementation
	}
}
