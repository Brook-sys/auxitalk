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
			Actions: []types.WorkflowAction{{Type: types.WorkflowActionEmitEvent, Risk: types.ActionRiskLow}},
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

	replacement := types.Workflow{
		ID:      "workflow-2",
		Enabled: true,
		Rules: []types.WorkflowRule{{
			ID:      "rule-2",
			Enabled: true,
			Trigger: types.WorkflowTrigger{EventType: "test.event"},
			Actions: []types.WorkflowAction{{Type: types.WorkflowActionEmitEvent, Risk: types.ActionRiskLow}},
		}},
	}
	if err := store.ReplaceWorkflows(ctx, []types.Workflow{replacement}); err != nil {
		t.Fatalf("replace workflows: %v", err)
	}
	workflows, err = store.ListWorkflows(ctx)
	if err != nil {
		t.Fatalf("list replacement workflows: %v", err)
	}
	if len(workflows) != 1 || workflows[0].ID != replacement.ID {
		t.Fatalf("unexpected replacement workflows: %+v", workflows)
	}

	session := types.Session{
		ID:        "session-1",
		Channel:   "whatsapp",
		State:     "active",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	if err := store.SaveSession(ctx, session); err != nil {
		t.Fatalf("save session: %v", err)
	}
	sessions, err := store.ListSessions(ctx, 10)
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if len(sessions) != 1 || sessions[0].ID != session.ID || sessions[0].State != "active" {
		t.Fatalf("unexpected sessions: %+v", sessions)
	}
	loaded, err := store.GetSession(ctx, "session-1")
	if err != nil || loaded.ID != session.ID {
		t.Fatalf("get session: %v (loaded=%+v)", err, loaded)
	}
}
