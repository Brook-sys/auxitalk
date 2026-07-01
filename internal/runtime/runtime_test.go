package runtime

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Brook-sys/auxitalk/internal/config"
	"github.com/Brook-sys/auxitalk/pkg/types"
)

func TestRuntimeLoadsEnabledPluginManifest(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "plugin.sh")
	if err := os.WriteFile(script, []byte(`#!/usr/bin/env sh
printf started > started.txt
printf '%s' "$AUXITALK_TEST_ENV" > env.txt
printf '{"jsonrpc":"2.0","id":"evt-1","method":"event.emit","params":{"type":"fake.started","payload":{"ok":true}}}\n'
printf '{"jsonrpc":"2.0","id":"act-1","method":"action.request","params":{"type":"message.send","risk":"high","payload":{"text":"hello"}}}\n'
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

	t.Setenv("AUXITALK_TEST_SECRET", "secret-value")
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
			Plugins: []config.Plugin{{
				Manifest: manifest,
				Enabled:  true,
				Env: map[string]string{
					"AUXITALK_TEST_ENV": "${AUXITALK_TEST_SECRET}",
				},
			}},
		},
	})

	events := make(chan types.Event, 1)
	sub, err := r.Events().Subscribe("fake.started", func(ctx context.Context, event types.Event) error {
		events <- event
		return nil
	})
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	defer sub.Unsubscribe()

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
	envValue, err := os.ReadFile(filepath.Join(dir, "env.txt"))
	if err != nil {
		t.Fatalf("read env file: %v", err)
	}
	if string(envValue) != "secret-value" {
		t.Fatalf("unexpected plugin env: %q", string(envValue))
	}

	select {
	case event := <-events:
		if event.Source != "fake-plugin" {
			t.Fatalf("unexpected event source: %s", event.Source)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("event was not published")
	}

	statuses := r.PluginStatuses()
	if len(statuses) != 1 {
		t.Fatalf("expected one plugin status, got %d", len(statuses))
	}
	if statuses[0].ID != "fake-plugin" || !statuses[0].Running {
		t.Fatalf("unexpected plugin status: %+v", statuses[0])
	}

	for i := 0; i < 50; i++ {
		if len(r.Actions()) == 1 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	actions := r.Actions()
	if len(actions) != 1 {
		t.Fatalf("expected one action, got %d", len(actions))
	}
	if actions[0].Source != "fake-plugin" || actions[0].Status != types.ActionStatusAllowed {
		t.Fatalf("unexpected action: %+v", actions[0])
	}
	denied, err := r.DenyAction(actions[0].ID)
	if err != nil {
		t.Fatalf("deny action: %v", err)
	}
	if denied.Status != types.ActionStatusDenied {
		t.Fatalf("expected denied action, got %+v", denied)
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
