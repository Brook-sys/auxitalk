package plugins

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Brook-sys/auxitalk/pkg/types"
)

func writeManifestFile(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "plugin.json")
	content := `{
		"id": "mock-input",
		"name": "Mock Input",
		"version": "0.1.0",
		"runtime": "node",
		"entry": "index.js",
		"kind": "input"
	}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	return path
}

func TestSupervisorStartStop(t *testing.T) {
	tmp := t.TempDir()
	writeManifestFile(t, tmp)

	manifest, err := LoadManifest(filepath.Join(tmp, "plugin.json"))
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}

	registry := NewRegistry()
	if err := registry.RegisterManifest(manifest); err != nil {
		t.Fatalf("register: %v", err)
	}

	plugin := registry.List()[0]
	sup := NewSupervisor(plugin, SupervisorOptions{
		HealthInterval: 100 * time.Millisecond,
		HealthTimeout:  50 * time.Millisecond,
		RestartBackoff: 10 * time.Millisecond,
		MaxRestarts:    1,
	})

	if err := sup.Start(context.Background()); err != nil {
		t.Fatalf("start supervisor: %v", err)
	}

	if !sup.IsRunning() {
		t.Fatal("expected supervisor running")
	}

	if err := sup.Stop(); err != nil {
		t.Fatalf("stop supervisor: %v", err)
	}

	if sup.IsRunning() {
		t.Fatal("expected supervisor stopped")
	}
}

func TestSupervisorRejectsInvalidManifest(t *testing.T) {
	plugin := Plugin{
		Manifest: types.PluginManifest{
			ID: "invalid",
		},
	}

	sup := NewSupervisor(plugin, SupervisorOptions{})
	if err := sup.Start(context.Background()); err == nil {
		t.Fatal("expected manifest validation error")
	}
}
