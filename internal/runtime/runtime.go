package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"github.com/Brook-sys/auxitalk/internal/actions"
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
	actions    *actions.Store
	gate       *actions.Gate
	supervisor *supervisor.Supervisor
}

func New(options Options) *Runtime {
	r := &Runtime{
		options: options,
		actions: actions.NewStore(),
		gate:    actions.NewGate(options.Config.Mode),
		events: events.New(events.Options{
			HandlerTimeout: options.Config.Runtime.RequestTimeout.Std(),
			HistoryLimit:   1000,
		}),
	}

	r.supervisor = supervisor.NewSupervisor(supervisor.ProcessOptions{
		CallTimeout:       options.Config.Runtime.RequestTimeout.Std(),
		HealthInterval:    30 * time.Second,
		RestartBackoff:    time.Second,
		MaxRestarts:       3,
		MaxHealthFailures: 3,
		MaxPayloadSize:    int(options.Config.Runtime.MaxPayloadSize),
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

func (r *Runtime) Actions() []types.ActionRequest {
	return r.actions.List()
}

func (r *Runtime) PendingActions() []types.ActionRequest {
	return r.actions.Pending()
}

func (r *Runtime) RecentEvents() []types.Event {
	return r.events.History()
}

func (r *Runtime) ApproveAction(id string) (types.ActionRequest, error) {
	action, err := r.actions.UpdateStatus(id, types.ActionStatusAllowed)
	if err != nil {
		return types.ActionRequest{}, err
	}
	r.publishActionStatus("action.approved", action)
	return action, nil
}

func (r *Runtime) DenyAction(id string) (types.ActionRequest, error) {
	action, err := r.actions.UpdateStatus(id, types.ActionStatusDenied)
	if err != nil {
		return types.ActionRequest{}, err
	}
	r.publishActionStatus("action.denied", action)
	return action, nil
}

func (r *Runtime) publishActionStatus(eventType string, action types.ActionRequest) {
	_ = r.events.Publish(context.Background(), types.Event{
		ID:        fmt.Sprintf("%s-%d", eventType, time.Now().UnixNano()),
		Type:      eventType,
		Source:    "core.runtime",
		SessionID: action.SessionID,
		CreatedAt: time.Now().UTC(),
		Payload: map[string]any{
			"id":     action.ID,
			"type":   action.Type,
			"risk":   action.Risk,
			"status": action.Status,
			"source": action.Source,
		},
	})
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
	case "action.request":
		err = r.handleActionRequest(req)
	case "plugin.stop":
		err = r.handlePluginStop(req)
	default:
		err = req.Reject(protocol.ErrMethodNotFound, "method not found")
	}
	if err != nil {
		fmt.Printf("[%s] request error %s: %v\n", req.PluginID, req.Method, err)
	}
}

func (r *Runtime) handleActionRequest(req supervisor.ProcessRequest) error {
	var action types.ActionRequest
	if err := json.Unmarshal(req.Params, &action); err != nil {
		return req.Reject(protocol.ErrInvalidParams, "invalid action params")
	}
	if action.Source == "" {
		action.Source = req.PluginID
	}
	if action.ID == "" {
		action.ID = fmt.Sprintf("%s-action-%d", req.PluginID, time.Now().UnixNano())
	}
	if action.Status == "" {
		action.Status = types.ActionStatusRequested
	}
	if action.CreatedAt.IsZero() {
		action.CreatedAt = time.Now().UTC()
	}
	status, err := r.gate.Decide(action)
	if err != nil && status == "" {
		return req.Reject(protocol.ErrInvalidParams, err.Error())
	}
	action.Status = status
	r.actions.Save(action)
	_ = r.events.Publish(context.Background(), types.Event{
		ID:        fmt.Sprintf("%s-event-%d", action.ID, time.Now().UnixNano()),
		Type:      "action.requested",
		Source:    "core.runtime",
		SessionID: action.SessionID,
		CreatedAt: time.Now().UTC(),
		Payload: map[string]any{
			"id":     action.ID,
			"type":   action.Type,
			"risk":   action.Risk,
			"status": action.Status,
			"source": action.Source,
		},
	})
	return req.Respond(action)
}

func (r *Runtime) handlePluginStop(req supervisor.ProcessRequest) error {
	id := req.PluginID
	_ = r.events.Publish(context.Background(), types.Event{
		ID:        fmt.Sprintf("%s-stop-%d", id, time.Now().UnixNano()),
		Type:      "plugin.stop.requested",
		Source:    "core.runtime",
		CreatedAt: time.Now().UTC(),
		Payload:   map[string]any{"id": id},
	})
	if err := r.supervisor.Stop(id); err != nil {
		return req.Reject(protocol.ErrInternal, err.Error())
	}
	return req.Respond(map[string]any{"ok": true})
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
			Env:     pluginConfig.ResolvedEnv(nil),
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
