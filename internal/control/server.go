package control

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Brook-sys/auxitalk/internal/plugins/supervisor"
	"github.com/Brook-sys/auxitalk/pkg/types"
)

type Runtime interface {
	PluginStatuses() []supervisor.ProcessStatus
	ConfiguredPlugins() []map[string]any
	RecentEvents() []types.Event
	Actions() []types.ActionRequest
	PendingActions() []types.ActionRequest
	Workflows() []types.Workflow
	Logs(limit int64) (string, error)
	ApproveAction(id string) (types.ActionRequest, error)
	DenyAction(id string) (types.ActionRequest, error)
	ReloadWorkflows(workflows []types.Workflow) error
	EnableWorkflow(id string) error
	DisableWorkflow(id string) error
	EmitEvent(ctx context.Context, event types.Event) error
	ConfigurePlugin(id string, enabled bool, env map[string]string) error
}

type Server struct {
	runtime Runtime
	server  *http.Server
}

func New(addr string, runtime Runtime) *Server {
	if addr == "" {
		addr = ":8090"
	}
	s := &Server{runtime: runtime}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.health)
	mux.HandleFunc("/api/status", s.status)
	mux.HandleFunc("/api/plugins", s.plugins)
	mux.HandleFunc("/api/plugins/configure", s.configurePlugin)
	mux.HandleFunc("/api/events", s.eventsRoute)
	mux.HandleFunc("/api/events/stream", s.eventsStream)
	mux.HandleFunc("/api/actions", s.actions)
	mux.HandleFunc("/api/actions/stream", s.actionsStream)
	mux.HandleFunc("/api/workflows", s.workflows)
	mux.HandleFunc("/api/logs", s.logs)
	mux.HandleFunc("/api/logs/stream", s.logStream)
	mux.HandleFunc("/api/actions/", s.actionMutation)
	mux.HandleFunc("/api/workflows/reload", s.reloadWorkflows)
	mux.HandleFunc("/api/workflows/", s.workflowMutation)
	s.server = &http.Server{Addr: addr, Handler: withCORS(mux)}
	return s
}

func (s *Server) Start() error {
	fmt.Printf("[control] listening on %s\n", s.server.Addr)
	return s.server.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]any{"ok": true, "service": "auxitalk-control"})
}

func (s *Server) status(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]any{
		"plugins":           s.runtime.PluginStatuses(),
		"configuredPlugins": s.runtime.ConfiguredPlugins(),
		"events":            s.runtime.RecentEvents(),
		"actions":           s.runtime.Actions(),
		"pendingActions":    s.runtime.PendingActions(),
		"workflows":         s.runtime.Workflows(),
	})
}

func (s *Server) plugins(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.runtime.PluginStatuses())
}

func (s *Server) configurePlugin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var payload struct {
		ID      string            `json:"id"`
		Enabled bool              `json:"enabled"`
		Env     map[string]string `json:"env"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.runtime.ConfigurePlugin(payload.ID, payload.Enabled, payload.Env); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, map[string]any{"ok": true})
}

func (s *Server) eventsRoute(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var event types.Event
		if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := s.runtime.EmitEvent(r.Context(), event); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, map[string]any{"ok": true, "id": event.ID})
		return
	}
	writeJSON(w, s.runtime.RecentEvents())
}

func (s *Server) eventsStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	var lastID string
	for {
		events := s.runtime.RecentEvents()
		if len(events) > 0 {
			latest := events[len(events)-1].ID
			if latest != lastID {
				data, _ := json.Marshal(events)
				fmt.Fprintf(w, "data: %s\n\n", data)
				flusher.Flush()
				lastID = latest
			}
		}
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			fmt.Fprintf(w, ": heartbeat\n\n")
			flusher.Flush()
		}
	}
}

func (s *Server) actions(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.runtime.Actions())
}

func (s *Server) actionsStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	var lastCount int
	for {
		actions := s.runtime.Actions()
		if len(actions) != lastCount {
			data, _ := json.Marshal(actions)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
			lastCount = len(actions)
		}
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			fmt.Fprintf(w, ": heartbeat\n\n")
			flusher.Flush()
		}
	}
}

func (s *Server) workflows(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.runtime.Workflows())
}

func (s *Server) logs(w http.ResponseWriter, r *http.Request) {
	limit, ok := parseLogLimit(w, r)
	if !ok {
		return
	}
	content, err := s.runtime.Logs(limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]any{"content": content})
}

func (s *Server) logStream(w http.ResponseWriter, r *http.Request) {
	limit, ok := parseLogLimit(w, r)
	if !ok {
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	last := ""
	for {
		content, err := s.runtime.Logs(limit)
		if err != nil {
			fmt.Fprintf(w, "event: error\ndata: %s\n\n", strings.ReplaceAll(err.Error(), "\n", " "))
			flusher.Flush()
			return
		}
		if content != last {
			data, _ := json.Marshal(content)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
			last = content
		}
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			fmt.Fprintf(w, ": heartbeat\n\n")
			flusher.Flush()
		}
	}
}

func parseLogLimit(w http.ResponseWriter, r *http.Request) (int64, bool) {
	limit := int64(64 * 1024)
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || parsed < 0 {
			http.Error(w, "invalid limit", http.StatusBadRequest)
			return 0, false
		}
		limit = parsed
	}
	return limit, true
}

func (s *Server) workflowMutation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/workflows/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 {
		http.NotFound(w, r)
		return
	}
	id := parts[0]
	operation := parts[1]
	switch operation {
	case "enable":
		if err := s.runtime.EnableWorkflow(id); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	case "disable":
		if err := s.runtime.DisableWorkflow(id); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	default:
		http.NotFound(w, r)
		return
	}
	writeJSON(w, map[string]any{"ok": true})
}

func (s *Server) reloadWorkflows(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var payload struct {
		Workflows []types.Workflow `json:"workflows"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.runtime.ReloadWorkflows(payload.Workflows); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, map[string]any{"ok": true})
}

func (s *Server) actionMutation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/actions/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 {
		http.NotFound(w, r)
		return
	}
	id := parts[0]
	operation := parts[1]
	var action types.ActionRequest
	var err error
	switch operation {
	case "approve":
		action, err = s.runtime.ApproveAction(id)
	case "deny":
		action, err = s.runtime.DenyAction(id)
	default:
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, action)
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
