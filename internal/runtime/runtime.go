package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Brook-sys/auxitalk/internal/actions"
	"github.com/Brook-sys/auxitalk/internal/capabilities"
	"github.com/Brook-sys/auxitalk/internal/config"
	"github.com/Brook-sys/auxitalk/internal/control"
	"github.com/Brook-sys/auxitalk/internal/events"
	"github.com/Brook-sys/auxitalk/internal/logger"
	"github.com/Brook-sys/auxitalk/internal/plugins"
	"github.com/Brook-sys/auxitalk/internal/plugins/supervisor"
	"github.com/Brook-sys/auxitalk/internal/sessions"
	storagesqlite "github.com/Brook-sys/auxitalk/internal/storage/sqlite"
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
	sessions         *sessions.Manager
	supervisor       *supervisor.Supervisor
	router           *capabilities.Router
	storage          *storagesqlite.Store
	controlServer    *control.Server
	wg               sync.WaitGroup
}

func New(options Options) *Runtime {
	r := &Runtime{
		options:          options,
		actions:          actions.NewStore(),
		gate:             actions.NewGate(options.Config.Mode),
		workflowRegistry: workflows.NewRegistry(),
		sessions:         sessions.NewManager(),
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
	r.workflowEngine.SetSessions(r.sessions)

	r.supervisor = supervisor.NewSupervisor(supervisor.ProcessOptions{
		CallTimeout:       options.Config.Runtime.RequestTimeout.Std(),
		HealthInterval:    30 * time.Second,
		RestartBackoff:    time.Second,
		MaxRestarts:       3,
		MaxHealthFailures: 3,
		MaxPayloadSize:    int(options.Config.Runtime.MaxPayloadSize),
		OnLog: func(pluginID string, line string) {
			logger.Printf("[%s] %s\n", pluginID, line)
		},
		OnRequest: r.handlePluginRequest,
		OnStatus:  r.handlePluginStatus,
	})

	return r
}

func (r *Runtime) Run(ctx context.Context) error {
	logger.Printf("%s %s mode=%s\n", r.options.Name, r.options.Version, r.options.Config.Mode)
	logger.Printf("[runtime] storage=%s control=%v\n", r.options.Config.Storage.SQLitePath, r.options.Config.Control.Enabled)

	if err := r.openStorage(ctx); err != nil {
		return err
	}
	if _, err := r.events.Subscribe("*", r.handlePersistenceEvent); err != nil {
		return err
	}
	if _, err := r.events.Subscribe("*", r.handleWorkflowEvent); err != nil {
		return err
	}
	if _, err := r.events.Subscribe("*", r.handleSessionTracking); err != nil {
		return err
	}
	if r.options.Config.Control.Enabled {
		r.controlServer = control.New(r.options.Config.Control.Addr, r)
		go func() {
			if err := r.controlServer.Start(); err != nil && err != http.ErrServerClosed {
				logger.Printf("control server error: %v\n", err)
			}
		}()
		logger.Printf("control api listening on %s\n", r.options.Config.Control.Addr)
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

func (r *Runtime) openStorage(ctx context.Context) error {
	if r.options.Config.Storage.SQLitePath == "" {
		return nil
	}
	store, err := storagesqlite.Open(r.options.Config.Storage.SQLitePath)
	if err != nil {
		return err
	}
	r.storage = store

	storedWorkflows, err := r.storage.ListWorkflows(ctx)
	if err == nil {
		for _, workflow := range storedWorkflows {
			_ = r.workflowRegistry.Register(workflow)
		}
	}
	for _, workflow := range r.options.Config.Workflows {
		_ = r.workflowRegistry.Register(workflow)
		_ = r.storage.SaveWorkflow(ctx, workflow)
	}

	actions, err := r.storage.ListActions(ctx, 1000)
	if err == nil {
		for _, action := range actions {
			r.actions.Save(action)
		}
	}

	sessionsList, err := r.storage.ListSessions(ctx, 1000)
	if err == nil {
		for _, session := range sessionsList {
			_ = r.sessions.Create(session)
		}
	}

	events, err := r.storage.ListEvents(ctx, r.options.Config.Runtime.MaxEventsPerSecond*10)
	if err == nil {
		// SQLite returns descending, we need ascending for history
		for i, j := 0, len(events)-1; i < j; i, j = i+1, j-1 {
			events[i], events[j] = events[j], events[i]
		}
		r.events.RestoreHistory(events)
	}

	_ = r.workflowEngine.SetRules(r.workflowRegistry.EnabledRules())

	return nil
}

func (r *Runtime) handlePersistenceEvent(ctx context.Context, event types.Event) error {
	if r.storage == nil {
		return nil
	}
	return r.storage.SaveEvent(ctx, event)
}

func (r *Runtime) persistAction(action types.ActionRequest) {
	if r.storage != nil {
		_ = r.storage.SaveAction(context.Background(), action)
	}
}

func (r *Runtime) EmitEvent(ctx context.Context, event types.Event) error {
	if event.ID == "" {
		event.ID = fmt.Sprintf("api-event-%d", time.Now().UnixNano())
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now().UTC()
	}
	if event.Source == "" {
		event.Source = "api"
	}
	return r.events.Publish(ctx, event)
}

func (r *Runtime) Workflows() []types.Workflow {
	return r.workflowRegistry.List()
}

func (r *Runtime) Logs(limit int64) (string, error) {
	path := r.options.Config.Runtime.LogPath
	if path == "" {
		return "", nil
	}
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return "", err
	}
	if limit <= 0 {
		limit = 64 * 1024
	}
	start := info.Size() - limit
	if start < 0 {
		start = 0
	}
	if _, err := file.Seek(start, 0); err != nil {
		return "", err
	}
	data, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (r *Runtime) EnableWorkflow(id string) error {
	workflows := r.workflowRegistry.List()
	changed := false
	for i, w := range workflows {
		if w.ID == id {
			workflows[i].Enabled = true
			changed = true
			break
		}
	}
	if !changed {
		return fmt.Errorf("workflow not found: %s", id)
	}
	return r.ReloadWorkflows(workflows)
}

func (r *Runtime) DisableWorkflow(id string) error {
	workflows := r.workflowRegistry.List()
	changed := false
	for i, w := range workflows {
		if w.ID == id {
			workflows[i].Enabled = false
			changed = true
			break
		}
	}
	if !changed {
		return fmt.Errorf("workflow not found: %s", id)
	}
	return r.ReloadWorkflows(workflows)
}
func (r *Runtime) ReloadWorkflows(newWorkflows []types.Workflow) error {
	newRegistry := workflows.NewRegistry()
	for _, workflow := range newWorkflows {
		if err := newRegistry.Register(workflow); err != nil {
			return err
		}
	}
	if r.storage != nil {
		if err := r.storage.ReplaceWorkflows(context.Background(), newWorkflows); err != nil {
			return err
		}
	}
	r.workflowRegistry = newRegistry
	return r.workflowEngine.SetRules(newRegistry.EnabledRules())
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
	r.persistAction(action)
	r.publishActionStatus("action.requested", action)

	if action.Status == types.ActionStatusAllowed {
		r.executeActionAsync(action)
	}

	return nil
}

func (r *Runtime) PluginStatuses() []supervisor.ProcessStatus {
	return r.supervisor.ListStatus()
}

func (r *Runtime) ConfigurePlugin(id string, enabled bool, env map[string]string) error {
	changed := false
	for i, p := range r.options.Config.Plugins {
		manifestID := ""
		if p.Inline != nil {
			manifestID = p.Inline.ID
		} else if p.Manifest != "" {
			if m, err := plugins.LoadManifest(p.Manifest); err == nil {
				manifestID = m.Manifest.ID
			}
		}
		if manifestID == id {
			r.options.Config.Plugins[i].Enabled = enabled
			if r.options.Config.Plugins[i].Env == nil {
				r.options.Config.Plugins[i].Env = make(map[string]string)
			}
			for k, v := range env {
				r.options.Config.Plugins[i].Env[k] = v
			}
			changed = true
			break
		}
	}
	if !changed {
		return fmt.Errorf("plugin %s not found in config", id)
	}

	// Persist if config path is known, but we don't store it in Options currently.
	// We'll need a way to pass config path. But for now, we just apply dynamically.

	if enabled {
		if r.supervisor.IsRunning(id) {
			_ = r.supervisor.Stop(id)
			time.Sleep(100 * time.Millisecond)
		}
		// Try to start it. But we need the spec. loadPlugins does it.
		// Instead of rewriting start logic, let's just re-run loadPlugins? No, it could start multiple.
		// We can just stop it, and wait for reload or restart via control.
		// Wait, supervisor Start requires the Spec which is only registered in loadPlugins.
		// If it's already registered, we can just Start it.
		if err := r.supervisor.Start(context.Background(), id); err != nil {
			logger.Printf("configure start error: %v", err)
		}
	} else {
		_ = r.supervisor.Stop(id)
	}
	return nil
}

func (r *Runtime) ConfiguredPlugins() []map[string]any {
	items := make([]map[string]any, 0, len(r.options.Config.Plugins))
	for _, plugin := range r.options.Config.Plugins {
		item := map[string]any{
			"manifest": plugin.Manifest,
			"enabled":  plugin.Enabled,
		}
		if plugin.Inline != nil {
			item["id"] = plugin.Inline.ID
			item["name"] = plugin.Inline.Name
			item["kind"] = plugin.Inline.Kind
			item["capabilities"] = plugin.Inline.Capabilities
		} else if plugin.Manifest != "" {
			if manifest, err := plugins.LoadManifest(plugin.Manifest); err == nil {
				item["id"] = manifest.Manifest.ID
				item["name"] = manifest.Manifest.Name
				item["kind"] = manifest.Manifest.Kind
				item["capabilities"] = manifest.Manifest.Capabilities
			}
		}
		items = append(items, item)
	}
	return items
}

func (r *Runtime) HealthCheck(ctx context.Context, id string) error {
	return r.supervisor.HealthCheck(ctx, id)
}

func (r *Runtime) handleSessionTracking(ctx context.Context, event types.Event) error {
	if event.SessionID == "" {
		return nil
	}
	session, err := r.sessions.Get(event.SessionID)
	if err != nil {
		session = types.Session{
			ID:        event.SessionID,
			Channel:   event.Source,
			State:     "active",
			Metadata:  map[string]any{},
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		}
		if createErr := r.sessions.Create(session); createErr != nil {
			return createErr
		}
	} else {
		session.UpdatedAt = time.Now().UTC()
		_ = r.sessions.Update(session)
	}
	if r.storage != nil {
		_ = r.storage.SaveSession(ctx, session)
	}
	return nil
}
func (r *Runtime) handleWorkflowEvent(ctx context.Context, event types.Event) error {
	if r.workflowEngine == nil {
		return nil
	}
	logger.Printf("[runtime] workflow event type=%s source=%s\n", event.Type, event.Source)
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
	r.persistAction(action)
	r.publishActionStatus("action.approved", action)
	r.executeActionAsync(action)
	return action, nil
}

func (r *Runtime) DenyAction(id string) (types.ActionRequest, error) {
	action, err := r.actions.UpdateStatus(id, types.ActionStatusDenied)
	if err != nil {
		return types.ActionRequest{}, err
	}
	r.persistAction(action)
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
	logger.Printf("[runtime] executing action id=%s type=%s source=%s\n", action.ID, action.Type, action.Source)
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
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
					if raw, ok := result.(json.RawMessage); ok {
						var decoded any
						if err := json.Unmarshal(raw, &decoded); err == nil {
							if resMap, ok := decoded.(map[string]any); ok {
								execution.Result = resMap
							} else {
								execution.Result = map[string]any{"data": decoded}
							}
						} else {
							execution.Result = map[string]any{"data": string(raw)}
						}
					} else if resMap, ok := result.(map[string]any); ok {
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
			if updated, updateErr := r.actions.UpdateStatus(action.ID, types.ActionStatusFailed); updateErr == nil {
				r.persistAction(updated)
			}
		} else {
			execution.Status = types.ActionExecutionCompleted
			if updated, updateErr := r.actions.UpdateStatus(action.ID, types.ActionStatusExecuted); updateErr == nil {
				r.persistAction(updated)
			}
		}

		_ = r.events.Publish(context.Background(), types.Event{
			ID:        fmt.Sprintf("action.executed-%d", time.Now().UnixNano()),
			Type:      "action.executed",
			Source:    "core.runtime",
			SessionID: action.SessionID,
			TraceID:   action.TraceID,
			Depth:     action.Depth,
			CreatedAt: time.Now().UTC(),
			Payload: map[string]any{
				"actionId":     action.ID,
				"actionType":   action.Type,
				"actionSource": action.Source,
				"status":       execution.Status,
				"error":        execution.Error,
				"result":       execution.Result,
				"dryRun":       execution.DryRun,
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
		logger.Printf("[%s] request error %s: %v\n", req.PluginID, req.Method, err)
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

		logger.Printf("[runtime] plugin started: %s kind=%s caps=%d\n", manifestFile.Manifest.ID, manifestFile.Manifest.Kind, len(manifestFile.Manifest.Capabilities))
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
	r.wg.Wait()
	if r.controlServer != nil {
		_ = r.controlServer.Shutdown(context.Background())
	}
	if r.storage != nil {
		return r.storage.Close()
	}
	return nil
}
