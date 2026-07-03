package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/Brook-sys/auxitalk/internal/actions"
	"github.com/Brook-sys/auxitalk/internal/capabilities"
	"github.com/Brook-sys/auxitalk/internal/config"
	"github.com/Brook-sys/auxitalk/internal/events"
	"github.com/Brook-sys/auxitalk/internal/plugins"
	"github.com/Brook-sys/auxitalk/internal/plugins/supervisor"
	"github.com/Brook-sys/auxitalk/internal/workflows"
	"github.com/Brook-sys/auxitalk/pkg/protocol"
	"github.com/Brook-sys/auxitalk/pkg/types"
)

type Options struct {
	Name    string
	Version string
	Config  config.Config
}

type Runtime struct {
	options          Options
	events           *events.Bus
	actions          *actions.Store
	gate             *actions.Gate
	workflowRegistry *workflows.Registry
	workflowEngine   *workflows.Engine
	supervisor       *supervisor.Supervisor
	router           *capabilities.Router
}

func New(options Options) *Runtime {
	r := &Runtime{
		options:          options,
		actions:          actions.NewStore(),
		gate:             actions.NewGate(options.Config.Mode),
		workflowRegistry: workflows.NewRegistry(),
		router:           capabilities.NewRouter(),
		events: events.New(events.Options{
			HandlerTimeout: options.Config.Runtime.RequestTimeout.Std(),
			HistoryLimit:   1000,
		}),
	}
	for _, workflow := range options.Config.Workflows {
		_ = r.workflowRegistry.Register(workflow)
	}
	r.workflowEngine, _ = workflows.NewEngine(r, r.workflowRegistry.EnabledRules())
	r.workflowEngine.SetExecutor(workflows.NewMockExecutor())

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

	if _, err := r.events.Subscribe("*", r.handleWorkflowEvent); err != nil {
		return err
	}

	if err := r.loadPlugins(ctx); err != nil {
		return err
	}

	go r.healthLoop(ctx)

	<-ctx.Done()
	return r.shutdown()
}

func (r *Runtime) Events() *events.Bus {
	return r.events
}

func (r *Runtime) Workflows() []types.Workflow {
	return r.workflowRegistry.List()
}

func (r *Runtime) RequestAction(ctx context.Context, action types.ActionRequest) error {
	if action.Source == "" {
		action.Source = "workflow"
	}
	if action.ID == "" {
		action.ID = fmt.Sprintf("workflow-action-%d", time.Now().UnixNano())
	}
	if action.Status == "" {
		action.Status = types.ActionStatusRequested
	}
	if action.CreatedAt.IsZero() {
		action.CreatedAt = time.Now().UTC()
	}
	status, err := r.gate.Decide(action)
	if err != nil && status == "" {
		return err
	}
	action.Status = status
	r.actions.Save(action)
	r.publishActionStatus("action.requested", action)

	if action.Status == types.ActionStatusAllowed {
		r.executeActionAsync(action)
	}

	return nil
}

func (r *Runtime) PluginStatuses() []supervisor.ProcessStatus {
	return r.supervisor.ListStatus()
}

func (r *Runtime) HealthCheck(ctx context.Context, id string) error {
	return r.supervisor.HealthCheck(ctx, id)
}

func (r *Runtime) handleWorkflowEvent(ctx context.Context, event types.Event) error {
	if r.workflowEngine == nil {
		return nil
	}
	_, err := r.workflowEngine.HandleEvent(ctx, event)
	return err
}

func (r *Runtime) healthLoop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	failures := map[string]int{}
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for _, id := range r.supervisor.List() {
				if err := r.supervisor.HealthCheck(ctx, id); err != nil {
					failures[id]++
					if failures[id] >= 3 {
						_ = r.events.Publish(context.Background(), types.Event{
							ID:        fmt.Sprintf("%s-health-%d", id, time.Now().UnixNano()),
							Type:      "plugin.error",
							Source:    "core.runtime",
							CreatedAt: time.Now().UTC(),
							Payload: map[string]any{
								"id":       id,
								"failures": failures[id],
								"error":    err.Error(),
							},
						})
						failures[id] = 0
					}
				} else {
					failures[id] = 0
				}
			}
		}
	}
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
	r.executeActionAsync(action)
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

func (r *Runtime) executeActionAsync(action types.ActionRequest) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), r.options.Config.Runtime.RequestTimeout.Std())
		defer cancel()

		var execution types.ActionExecution
		var err error

		if types.IsWorkflowActionType(action.Type) {
			execution, err = r.workflowEngine.ExecuteAction(ctx, action)
		} else {
			// Capability routing fallback
			parts := strings.SplitN(action.Type, ".", 2)
			if len(parts) == 2 {
				pluginID := parts[0]
				capability := parts[1]
				result, callErr := r.router.Call(ctx, pluginID, capability, action.Payload)
				err = callErr
				execution = types.ActionExecution{
					ID:          fmt.Sprintf("exec-%s-%d", action.ID, time.Now().UnixNano()),
					ActionID:    action.ID,
					Type:        action.Type,
					Status:      types.ActionExecutionCompleted,
					Input:       action.Payload,
					Result:      nil,
					CreatedAt:   time.Now().UTC(),
					CompletedAt: time.Now().UTC(),
				}
				if result != nil {
					if resMap, ok := result.(map[string]any); ok {
						execution.Result = resMap
					} else {
						execution.Result = map[string]any{"data": result}
					}
				}
			} else {
				err = fmt.Errorf("unknown action type: %s", action.Type)
			}
		}

		if err != nil {
			execution.Status = types.ActionExecutionFailed
			execution.Error = err.Error()
			_, _ = r.actions.UpdateStatus(action.ID, types.ActionStatusFailed)
		} else {
			execution.Status = types.ActionExecutionCompleted
			_, _ = r.actions.UpdateStatus(action.ID, types.ActionStatusExecuted)
		}

		_ = r.events.Publish(context.Background(), types.Event{
			ID:        fmt.Sprintf("action.executed-%d", time.Now().UnixNano()),
			Type:      "action.executed",
			Source:    "core.runtime",
			SessionID: action.SessionID,
			CreatedAt: time.Now().UTC(),
			Payload: map[string]any{
				"actionId": action.ID,
				"status":   execution.Status,
				"error":    execution.Error,
				"result":   execution.Result,
				"dryRun":   execution.DryRun,
			},
		})
	}()
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
	if action.Status == types.ActionStatusAllowed {
		r.executeActionAsync(action)
	}
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

		pluginID := manifestFile.Manifest.ID
		for _, cap := range manifestFile.Manifest.Capabilities {
			capName := cap.Name
			err := r.router.RegisterCapability(pluginID, manifestFile.Manifest, capName, func(ctx context.Context, params any) (any, error) {
				return r.supervisor.Call(ctx, pluginID, "capability.call", map[string]any{
					"name":  capName,
					"input": params,
				})
			})
			if err != nil {
				return err
			}
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
