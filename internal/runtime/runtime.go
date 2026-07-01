package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"github.com/Brook-sys/auxitalk/internal/config"
	"github.com/Brook-sys/auxitalk/internal/events"
	"github.com/Brook-sys/auxitalk/internal/plugins"
	"github.com/Brook-sys/auxitalk/internal/plugins/supervisor"
	"github.com/Brook-sys/auxitalk/pkg/protocol"
	"github.com/Brook-sys/auxitalk/pkg/types"
)

type Options struct {
	Name    string
	Version string
	Config  config.Config
}

type Runtime struct {
	options    Options
	events     *events.Bus
	supervisor *supervisor.Supervisor
}

func New(options Options) *Runtime {
	r := &Runtime{
		options: options,
		events: events.New(events.Options{
			HandlerTimeout: options.Config.Runtime.RequestTimeout.Std(),
			HistoryLimit:   1000,
		}),
	}

	r.supervisor = supervisor.NewSupervisor(supervisor.ProcessOptions{
		CallTimeout:    options.Config.Runtime.RequestTimeout.Std(),
		HealthInterval: 30 * time.Second,
		RestartBackoff: time.Second,
		MaxRestarts:    3,
		MaxPayloadSize: int(options.Config.Runtime.MaxPayloadSize),
		OnLog: func(pluginID string, line string) {
			fmt.Printf("[%s] %s\n", pluginID, line)
		},
		OnRequest: r.handlePluginRequest,
		OnStatus:  r.handlePluginStatus,
	})

	return r
}

func (r *Runtime) Run(ctx context.Context) error {
	fmt.Printf("%s %s mode=%s\n", r.options.Name, r.options.Version, r.options.Config.Mode)

	if err := r.loadPlugins(ctx); err != nil {
		return err
	}

	<-ctx.Done()
	return r.shutdown()
}

func (r *Runtime) Events() *events.Bus {
	return r.events
}

func (r *Runtime) PluginStatuses() []supervisor.ProcessStatus {
	return r.supervisor.ListStatus()
}

func (r *Runtime) handlePluginStatus(status supervisor.ProcessStatus) {
	eventType := "plugin.stopped"
	if status.Running {
		eventType = "plugin.started"
	} else if status.LastError != "" {
		eventType = "plugin.error"
	}
	_ = r.events.Publish(context.Background(), types.Event{
		ID:        fmt.Sprintf("%s-status-%d", status.ID, time.Now().UnixNano()),
		Type:      eventType,
		Source:    "core.runtime",
		CreatedAt: time.Now().UTC(),
		Payload: map[string]any{
			"id":        status.ID,
			"running":   status.Running,
			"restarts":  status.Restarts,
			"lastError": status.LastError,
		},
	})
}

func (r *Runtime) handlePluginRequest(req supervisor.ProcessRequest) {
	var err error
	switch req.Method {
	case "event.emit":
		err = r.handleEventEmit(req)
	default:
		err = req.Reject(protocol.ErrMethodNotFound, "method not found")
	}
	if err != nil {
		fmt.Printf("[%s] request error %s: %v\n", req.PluginID, req.Method, err)
	}
}

func (r *Runtime) handleEventEmit(req supervisor.ProcessRequest) error {
	var event types.Event
	if err := json.Unmarshal(req.Params, &event); err != nil {
		return req.Reject(protocol.ErrInvalidParams, "invalid event params")
	}
	if event.Source == "" {
		event.Source = req.PluginID
	}
	if event.ID == "" {
		event.ID = fmt.Sprintf("%s-%d", req.PluginID, time.Now().UnixNano())
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now().UTC()
	}
	if err := r.events.Publish(context.Background(), event); err != nil {
		return req.Reject(protocol.ErrInvalidParams, err.Error())
	}
	return req.Respond(map[string]any{"ok": true})
}

func (r *Runtime) loadPlugins(ctx context.Context) error {
	for _, pluginConfig := range r.options.Config.Plugins {
		if !pluginConfig.Enabled {
			continue
		}
		if pluginConfig.Manifest == "" {
			continue
		}

		manifestFile, err := plugins.LoadManifest(pluginConfig.Manifest)
		if err != nil {
			return err
		}

		command := manifestFile.Manifest.Runtime
		args := []string{}

		if manifestFile.Manifest.Entry != "" {
			entry := manifestFile.Manifest.Entry
			if !filepath.IsAbs(entry) {
				entry = filepath.Join(manifestFile.Dir, entry)
			}
			args = append(args, entry)
		} else if !filepath.IsAbs(command) {
			command = filepath.Join(manifestFile.Dir, command)
		}

		spec := supervisor.ProcessSpec{
			ID:      manifestFile.Manifest.ID,
			Command: command,
			Args:    args,
			Dir:     manifestFile.Dir,
		}

		if err := r.supervisor.Register(spec); err != nil {
			return err
		}
		if err := r.supervisor.Start(ctx, manifestFile.Manifest.ID); err != nil {
			return err
		}

		fmt.Printf("plugin started: %s\n", manifestFile.Manifest.ID)
	}
	return nil
}

func (r *Runtime) shutdown() error {
	for _, id := range r.supervisor.List() {
		_, _ = r.supervisor.Call(context.Background(), id, "plugin.stop", nil)
		if err := r.supervisor.Stop(id); err != nil {
			fmt.Printf("plugin stop error %s: %v\n", id, err)
		}
	}
	return nil
}
