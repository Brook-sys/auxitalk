package workflows

import (
	"context"
	"testing"
	"time"

	"github.com/Brook-sys/auxitalk/pkg/types"
)

type memorySink struct {
	actions []types.ActionRequest
}

func (s *memorySink) RequestAction(ctx context.Context, action types.ActionRequest) error {
	s.actions = append(s.actions, action)
	return nil
}

func TestEngineHandlesEventAndRequestsAction(t *testing.T) {
	sink := &memorySink{}
	engine, err := NewEngine(sink, []types.WorkflowRule{{
		ID:      "auto-reply",
		Enabled: true,
		Trigger: types.WorkflowTrigger{EventType: "message.received", Source: "whatsapp"},
		Action: types.WorkflowAction{
			Type: "message.reply.suggest",
			Risk: types.ActionRiskMedium,
			Payload: map[string]any{
				"agent": "support",
			},
		},
	}})
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}

	event := types.Event{
		ID:        "event-1",
		Type:      "message.received",
		Source:    "whatsapp",
		SessionID: "session-1",
		CreatedAt: time.Now(),
	}
	actions, err := engine.HandleEvent(context.Background(), event)
	if err != nil {
		t.Fatalf("handle event: %v", err)
	}
	if len(actions) != 1 || len(sink.actions) != 1 {
		t.Fatalf("expected one action, got actions=%d sink=%d", len(actions), len(sink.actions))
	}
	action := actions[0]
	if action.Type != "message.reply.suggest" || action.Source != "workflow:auto-reply" || action.SessionID != "session-1" {
		t.Fatalf("unexpected action: %+v", action)
	}
	if action.Payload["eventId"] != "event-1" || action.Payload["agent"] != "support" {
		t.Fatalf("unexpected payload: %+v", action.Payload)
	}
}

func TestEngineIgnoresNonMatchingEvent(t *testing.T) {
	sink := &memorySink{}
	engine, err := NewEngine(sink, []types.WorkflowRule{{
		ID:      "terminal-command",
		Enabled: true,
		Trigger: types.WorkflowTrigger{EventType: "terminal.output"},
		Action:  types.WorkflowAction{Type: "terminal.command.suggest", Risk: types.ActionRiskHigh},
	}})
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}

	actions, err := engine.HandleEvent(context.Background(), types.Event{
		ID:        "event-1",
		Type:      "message.received",
		Source:    "whatsapp",
		CreatedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("handle event: %v", err)
	}
	if len(actions) != 0 || len(sink.actions) != 0 {
		t.Fatalf("expected no actions, got actions=%d sink=%d", len(actions), len(sink.actions))
	}
}
