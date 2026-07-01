package capabilities

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/Brook-sys/auxitalk/pkg/types"
)

var (
	ErrCapabilityNotFound = errors.New("capability not found")
	ErrPermissionDenied   = errors.New("permission denied")
)

type CapabilityHandler func(ctx context.Context, params any) (any, error)

type CapabilityInfo struct {
	PluginID string
	Manifest types.PluginManifest
	Handler  CapabilityHandler
}

type Router struct {
	mu           sync.RWMutex
	capabilities map[string]CapabilityInfo
}

func NewRouter() *Router {
	return &Router{
		capabilities: make(map[string]CapabilityInfo),
	}
}

func (r *Router) Register(pluginID string, manifest types.PluginManifest, handler CapabilityHandler) error {
	if err := manifest.Validate(); err != nil {
		return fmt.Errorf("invalid manifest: %w", err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	for _, cap := range manifest.Capabilities {
		name := cap.Name
		if _, exists := r.capabilities[name]; exists {
			return fmt.Errorf("capability %s already registered", name)
		}
		r.capabilities[name] = CapabilityInfo{
			PluginID: pluginID,
			Manifest: manifest,
			Handler:  handler,
		}
	}

	return nil
}

func (r *Router) Call(ctx context.Context, pluginID string, capability string, params any) (any, error) {
	r.mu.RLock()
	info, ok := r.capabilities[capability]
	r.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrCapabilityNotFound, capability)
	}

	if info.PluginID != pluginID {
		return nil, fmt.Errorf("%w: plugin %s does not own %s", ErrPermissionDenied, pluginID, capability)
	}

	if info.Handler == nil {
		return nil, fmt.Errorf("capability %s has no handler", capability)
	}

	return info.Handler(ctx, params)
}

func (r *Router) HasCapability(capability string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, ok := r.capabilities[capability]
	return ok
}

func (r *Router) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]string, 0, len(r.capabilities))
	for name := range r.capabilities {
		result = append(result, name)
	}
	return result
}
