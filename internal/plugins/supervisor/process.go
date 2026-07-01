package supervisor

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Brook-sys/auxitalk/pkg/protocol"
)

var (
	ErrAlreadyRunning = errors.New("plugin already running")
	ErrNotRunning     = errors.New("plugin is not running")
)

type ProcessSpec struct {
	ID      string
	Command string
	Args    []string
	Dir     string
	Env     []string
}

type ProcessOptions struct {
	CallTimeout    time.Duration
	HealthInterval time.Duration
	RestartBackoff time.Duration
	MaxRestarts    int
	MaxPayloadSize int
	OnRequest      func(ProcessRequest)
	OnLog          func(pluginID string, line string)
	OnStatus       func(ProcessStatus)
}

type ProcessStatus struct {
	ID        string        `json:"id"`
	Running   bool          `json:"running"`
	StartedAt time.Time     `json:"startedAt,omitempty"`
	StoppedAt time.Time     `json:"stoppedAt,omitempty"`
	Uptime    time.Duration `json:"uptime"`
	Restarts  int           `json:"restarts"`
	LastError string        `json:"lastError,omitempty"`
}

type ProcessRequest struct {
	PluginID string
	ID       string
	Method   string
	Params   json.RawMessage
	Respond  func(any) error
	Reject   func(code int, message string) error
}

type PluginProcess struct {
	spec      ProcessSpec
	opts      ProcessOptions
	cmd       *exec.Cmd
	stdin     io.WriteCloser
	stdout    io.ReadCloser
	stderr    io.ReadCloser
	mu        sync.Mutex
	running   bool
	seq       atomic.Uint64
	pending   map[string]chan protocol.Response
	ctx       context.Context
	cancel    context.CancelFunc
	restarts  int
	startedAt time.Time
	stoppedAt time.Time
	lastError error
}

func NewPluginProcess(spec ProcessSpec, opts ProcessOptions) *PluginProcess {
	if opts.CallTimeout <= 0 {
		opts.CallTimeout = 10 * time.Second
	}
	if opts.HealthInterval <= 0 {
		opts.HealthInterval = 30 * time.Second
	}
	if opts.RestartBackoff <= 0 {
		opts.RestartBackoff = time.Second
	}
	if opts.MaxPayloadSize <= 0 {
		opts.MaxPayloadSize = 1024 * 1024
	}
	return &PluginProcess{spec: spec, opts: opts, pending: map[string]chan protocol.Response{}}
}

func (p *PluginProcess) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.running {
		return ErrAlreadyRunning
	}

	return p.startLocked(ctx)
}

func (p *PluginProcess) startLocked(ctx context.Context) error {
	processCtx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(processCtx, p.spec.Command, p.spec.Args...)
	cmd.Dir = p.spec.Dir
	cmd.Env = append(cmd.Env, p.spec.Env...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	p.ctx, p.cancel = processCtx, cancel
	p.cmd = cmd
	p.stdin = stdin
	p.stdout = stdout
	p.stderr = stderr
	p.running = true
	p.startedAt = time.Now().UTC()
	p.stoppedAt = time.Time{}
	p.lastError = nil

	go p.readStdout()
	go p.readStderr()
	go p.monitor()
	go p.healthLoop()
	p.emitStatusLocked()

	return nil
}

func (p *PluginProcess) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return ErrNotRunning
	}

	p.stopLocked()
	return nil
}

func (p *PluginProcess) Call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	p.mu.Lock()
	if !p.running {
		p.mu.Unlock()
		return nil, ErrNotRunning
	}
	id := fmt.Sprintf("%s-%d", p.spec.ID, p.seq.Add(1))
	responseCh := make(chan protocol.Response, 1)
	p.pending[id] = responseCh
	stdin := p.stdin
	p.mu.Unlock()

	req, err := protocol.NewRequest(id, method, params)
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	if len(data)+1 > p.opts.MaxPayloadSize {
		return nil, fmt.Errorf("payload too large")
	}
	if _, err := stdin.Write(append(data, '\n')); err != nil {
		return nil, err
	}

	callCtx, cancel := context.WithTimeout(ctx, p.opts.CallTimeout)
	defer cancel()

	select {
	case <-callCtx.Done():
		p.deletePending(id)
		return nil, callCtx.Err()
	case resp := <-responseCh:
		if resp.Error != nil {
			return nil, resp.Error
		}
		return resp.Result, nil
	}
}

func (p *PluginProcess) IsRunning() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.running
}

func (p *PluginProcess) LastError() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.lastError
}

func (p *PluginProcess) Status() ProcessStatus {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.statusLocked()
}

func (p *PluginProcess) statusLocked() ProcessStatus {
	status := ProcessStatus{
		ID:        p.spec.ID,
		Running:   p.running,
		StartedAt: p.startedAt,
		StoppedAt: p.stoppedAt,
		Restarts:  p.restarts,
	}
	if p.running && !p.startedAt.IsZero() {
		status.Uptime = time.Since(p.startedAt)
	}
	if p.lastError != nil {
		status.LastError = p.lastError.Error()
	}
	return status
}

func (p *PluginProcess) emitStatusLocked() {
	if p.opts.OnStatus != nil {
		p.opts.OnStatus(p.statusLocked())
	}
}

func (p *PluginProcess) stopLocked() {
	if p.cancel != nil {
		p.cancel()
	}
	if p.cmd != nil && p.cmd.Process != nil {
		_ = p.cmd.Process.Kill()
	}
	p.running = false
	p.stoppedAt = time.Now().UTC()
	p.emitStatusLocked()
}

func (p *PluginProcess) readStdout() {
	scanner := bufio.NewScanner(p.stdout)
	buffer := make([]byte, 0, 64*1024)
	scanner.Buffer(buffer, p.opts.MaxPayloadSize)
	for scanner.Scan() {
		p.handleMessage(scanner.Bytes())
	}
}

func (p *PluginProcess) readStderr() {
	scanner := bufio.NewScanner(p.stderr)
	for scanner.Scan() {
		if p.opts.OnLog != nil {
			p.opts.OnLog(p.spec.ID, scanner.Text())
		}
	}
}

func (p *PluginProcess) handleMessage(data []byte) {
	var envelope struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      string          `json:"id,omitempty"`
		Method  string          `json:"method,omitempty"`
		Params  json.RawMessage `json:"params,omitempty"`
		Result  json.RawMessage `json:"result,omitempty"`
		Error   *protocol.Error `json:"error,omitempty"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return
	}
	if envelope.Method != "" {
		if p.opts.OnRequest != nil {
			p.opts.OnRequest(ProcessRequest{
				PluginID: p.spec.ID,
				ID:       envelope.ID,
				Method:   envelope.Method,
				Params:   envelope.Params,
				Respond: func(result any) error {
					return p.writeResponse(protocol.NewResponse(envelope.ID, result))
				},
				Reject: func(code int, message string) error {
					return p.writeResponse(protocol.NewError(envelope.ID, code, message), nil)
				},
			})
		}
		return
	}

	p.mu.Lock()
	ch := p.pending[envelope.ID]
	delete(p.pending, envelope.ID)
	p.mu.Unlock()
	if ch != nil {
		ch <- protocol.Response{JSONRPC: protocol.Version, ID: envelope.ID, Result: envelope.Result, Error: envelope.Error}
	}
}

func (p *PluginProcess) writeResponse(resp protocol.Response, err error) error {
	if err != nil {
		return err
	}
	data, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	if len(data)+1 > p.opts.MaxPayloadSize {
		return fmt.Errorf("payload too large")
	}

	p.mu.Lock()
	stdin := p.stdin
	running := p.running
	p.mu.Unlock()
	if !running {
		return ErrNotRunning
	}
	_, err = stdin.Write(append(data, '\n'))
	return err
}

func (p *PluginProcess) monitor() {
	err := p.cmd.Wait()

	p.mu.Lock()
	p.running = false
	p.stoppedAt = time.Now().UTC()
	p.lastError = err
	shouldRestart := p.ctx != nil && p.ctx.Err() == nil && p.restarts < p.opts.MaxRestarts
	if shouldRestart {
		p.restarts++
	}
	p.emitStatusLocked()
	p.mu.Unlock()

	if shouldRestart {
		time.Sleep(p.opts.RestartBackoff)
		p.mu.Lock()
		_ = p.startLocked(p.ctx)
		p.mu.Unlock()
	}
}

func (p *PluginProcess) healthLoop() {
	ticker := time.NewTicker(p.opts.HealthInterval)
	defer ticker.Stop()
	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			_, err := p.Call(p.ctx, "plugin.health", nil)
			if err != nil {
				p.mu.Lock()
				p.lastError = err
				p.emitStatusLocked()
				p.mu.Unlock()
			}
		}
	}
}

func (p *PluginProcess) deletePending(id string) {
	p.mu.Lock()
	delete(p.pending, id)
	p.mu.Unlock()
}
