package control

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

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
	ApproveAction(id string) (types.ActionRequest, error)
	DenyAction(id string) (types.ActionRequest, error)
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
	mux.HandleFunc("/api/events", s.events)
	mux.HandleFunc("/api/actions", s.actions)
	mux.HandleFunc("/api/workflows", s.workflows)
	mux.HandleFunc("/api/actions/", s.actionMutation)
	s.server = &http.Server{Addr: addr, Handler: withCORS(mux)}
	return s
}

func (s *Server) Start() error {
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

func (s *Server) events(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.runtime.RecentEvents())
}

func (s *Server) actions(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.runtime.Actions())
}

func (s *Server) workflows(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.runtime.Workflows())
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
