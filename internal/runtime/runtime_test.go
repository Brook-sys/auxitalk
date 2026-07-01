package runtime

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Brook-sys/auxitalk/internal/config"
)

func TestRuntimeLoadsEnabledPluginManifest(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "plugin.sh")
	if err := os.WriteFile(script, []byte(`#!/usr/bin/env sh
printf started > started.txt
while IFS= read -r line; do
  id=$(printf '%s' "$line" | sed -n 's/.*"id":"\([^"]*\)".*/\1/p')
  method=$(printf '%s' "$line" | sed -n 's/.*"method":"\([^"]*\)".*/\1/p')
  if [ "$method" = "plugin.stop" ]; then
    printf stopped > stopped.txt
    printf '{"jsonrpc":"2.0","id":"%s","result":{"ok":true}}\n' "$id"
    exit 0
  fi
  printf '{"jsonrpc":"2.0","id":"%s","result":{"ok":true}}\n' "$id"
done
`), 0o700); err != nil {
		t.Fatalf("write script: %v", err)
	}

	manifest := filepath.Join(dir, "plugin.json")
	if err := os.WriteFile(manifest, []byte(`{
  "id": "fake-plugin",
  "name": "Fake Plugin",
  "version": "0.1.0",
  "runtime": "sh",
  "entry": "./plugin.sh",
  "kind": "tool",
  "capabilities": []
}`), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	r := New(Options{
		Name:    "test",
		Version: "dev",
		Config: config.Config{
			Mode: config.ModeDev,
			Runtime: config.Runtime{
				RequestTimeout:     config.Duration(time.Second),
				HealthTimeout:      config.Duration(time.Second),
				MaxPayloadSize:     1024 * 1024,
				MaxEventsPerSecond: 50,
			},
			Plugins: []config.Plugin{{Manifest: manifest, Enabled: true}},
		},
	})

	done := make(chan error, 1)
	go func() { done <- r.Run(ctx) }()

	for i := 0; i < 50; i++ {
		if _, err := os.Stat(filepath.Join(dir, "started.txt")); err == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if _, err := os.Stat(filepath.Join(dir, "started.txt")); err != nil {
		t.Fatalf("plugin did not start: %v", err)
	}

	cancel()
	select {
	case err := <-done:
		if err != nil && err != context.Canceled {
			t.Fatalf("run: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("runtime did not stop")
	}
	if _, err := os.Stat(filepath.Join(dir, "stopped.txt")); err != nil {
		t.Fatalf("plugin did not stop gracefully: %v", err)
	}
}
