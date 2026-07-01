package capabilities

import (
	"context"
	"errors"
	"testing"

	"github.com/Brook-sys/auxitalk/pkg/types"
)

func testManifest() types.PluginManifest {
	return types.PluginManifest{
		ID:      "mock-ai",
		Name:    "Mock AI",
		Version: "0.1.0",
		Runtime: "node",
		Entry:   "index.js",
		Kind:    types.PluginKindAI,
		Capabilities: []types.Capability{
			{Name: "ai.complete"},
		},
	}
}

func TestRouterRegisterAndCall(t *testing.T) {
	router := NewRouter()
	manifest := testManifest()

	called := false
	handler := func(_ context.Context, params any) (any, error) {
		called = true
		return "ok", nil
	}

	if err := router.Register("mock-ai", manifest, handler); err != nil {
		t.Fatalf("register: %v", err)
	}

	result, err := router.Call(context.Background(), "mock-ai", "ai.complete", nil)
	if err != nil {
		t.Fatalf("call: %v", err)
	}
	if result != "ok" {
		t.Fatalf("unexpected result: %v", result)
	}
	if !called {
		t.Fatal("handler was not called")
	}
}

func TestRouterRejectsDuplicateCapability(t *testing.T) {
	router := NewRouter()
	manifest := testManifest()

	if err := router.Register("mock-ai", manifest, nil); err != nil {
		t.Fatalf("first register: %v", err)
	}

	err := router.Register("mock-ai", manifest, nil)
	if err == nil {
		t.Fatal("expected duplicate capability error")
	}
}

func TestRouterPermissionDenied(t *testing.T) {
	router := NewRouter()
	manifest := testManifest()

	if err := router.Register("mock-ai", manifest, nil); err != nil {
		t.Fatalf("register: %v", err)
	}

	_, err := router.Call(context.Background(), "other-plugin", "ai.complete", nil)
	if !errors.Is(err, ErrPermissionDenied) {
		t.Fatalf("expected permission denied, got %v", err)
	}
}

func TestRouterCapabilityNotFound(t *testing.T) {
	router := NewRouter()

	_, err := router.Call(context.Background(), "mock-ai", "unknown.capability", nil)
	if !errors.Is(err, ErrCapabilityNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestRouterList(t *testing.T) {
	router := NewRouter()
	manifest := testManifest()

	if err := router.Register("mock-ai", manifest, nil); err != nil {
		t.Fatalf("register: %v", err)
	}

	list := router.List()
	if len(list) != 1 || list[0] != "ai.complete" {
		t.Fatalf("unexpected list: %v", list)
	}
}
