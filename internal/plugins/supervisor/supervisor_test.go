package supervisor

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSupervisorStartsAndCallsHealth(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "plugin.sh")
	content := `#!/usr/bin/env sh
while IFS= read -r line; do
  id=$(printf '%s' "$line" | sed -n 's/.*"id":"\([^"]*\)".*/\1/p')
  method=$(printf '%s' "$line" | sed -n 's/.*"method":"\([^"]*\)".*/\1/p')
  if [ "$method" = "plugin.health" ]; then
    printf '{"jsonrpc":"2.0","id":"%s","result":{"ok":true}}\n' "$id"
  else
    printf '{"jsonrpc":"2.0","id":"%s","result":{"ok":true}}\n' "$id"
  fi
done
`
	if err := os.WriteFile(script, []byte(content), 0o700); err != nil {
		t.Fatalf("write script: %v", err)
	}

	sup := NewSupervisor(ProcessOptions{CallTimeout: time.Second, HealthInterval: time.Hour})
	if err := sup.Register(ProcessSpec{ID: "fake", Command: script}); err != nil {
		t.Fatalf("register: %v", err)
	}
	if err := sup.Start(context.Background(), "fake"); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer sup.Stop("fake")

	result, err := sup.Call(context.Background(), "fake", "plugin.health", nil)
	if err != nil {
		t.Fatalf("call: %v", err)
	}
	var parsed struct {
		OK bool `json:"ok"`
	}
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !parsed.OK {
		t.Fatal("expected ok")
	}
}
