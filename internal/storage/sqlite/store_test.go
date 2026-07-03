package sqlite

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/Brook-sys/auxitalk/pkg/types"
)

func TestStorePersistsEventsActionsAndWorkflows(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "auxitalk.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	event := types.Event{
		ID:        "event-1",
		Type:      "message.received",
		Source:    "test",
		SessionID: "session-1",
		Payload:   map[string]any{"text": "hello"},
		CreatedAt: time.Now().UTC(),
	}
	if err := store.SaveEvent(ctx, event); err != nil {
		t.Fatalf("save event: %v", err)
	}
	events, err := store.ListEvents(ctx, 10)
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if len(events) != 1 || events[0].ID != event.ID || events[0].Payload["text"] != "hello" {
		t.Fatalf("unexpected events: %+v", events)
	}

	action := types.ActionRequest{
		ID:        "action-1",
		Type:      types.WorkflowActionEmitEvent,
		Risk:      types.ActionRiskLow,
		Status:    types.ActionStatusAllowed,
		Source:    "test",
		SessionID: "session-1",
		Payload:   map[string]any{"eventType": "test.event"},
		CreatedAt: time.Now().UTC(),
	}
	if err := store.SaveAction(ctx, action); err != nil {
		t.Fatalf("save action: %v", err)
	}
	actions, err := store.ListActions(ctx, 10)
	if err != nil {
		t.Fatalf("list actions: %v", err)
	}
	if len(actions) != 1 || actions[0].ID != action.ID || actions[0].Payload["eventType"] != "test.event" {
		t.Fatalf("unexpected actions: %+v", actions)
	}

	workflow := types.Workflow{
		ID:      "workflow-1",
		Enabled: true,
		Rules: []types.WorkflowRule{{
			ID:      "rule-1",
			Enabled: true,
			Trigger: types.WorkflowTrigger{EventType: "message.received"},
			Action:  types.WorkflowAction{Type: types.WorkflowActionEmitEvent, Risk: types.ActionRiskLow},
		}},
	}
	if err := store.SaveWorkflow(ctx, workflow); err != nil {
		t.Fatalf("save workflow: %v", err)
	}
	workflows, err := store.ListWorkflows(ctx)
	if err != nil {
		t.Fatalf("list workflows: %v", err)
	}
	if len(workflows) != 1 || workflows[0].ID != workflow.ID {
		t.Fatalf("unexpected workflows: %+v", workflows)
	}
}
