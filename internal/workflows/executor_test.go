package workflows

import (
	"context"
	"testing"
	"time"

	"github.com/Brook-sys/auxitalk/pkg/types"
)

func TestMockExecutorSimulatesSupportedActions(t *testing.T) {
	executor := NewMockExecutor()
	for _, actionType := range []string{
		types.WorkflowActionSendMessage,
		types.WorkflowActionRunCommand,
		types.WorkflowActionCallPlugin,
		types.WorkflowActionEmitEvent,
	} {
		action := types.ActionRequest{
			ID:        "action-" + actionType,
			Type:      actionType,
			Risk:      types.ActionRiskLow,
			Status:    types.ActionStatusAllowed,
			Source:    "test",
			Payload:   map[string]any{"value": actionType},
			CreatedAt: time.Now(),
		}
		execution, err := executor.Execute(context.Background(), action)
		if err != nil {
			t.Fatalf("execute %s: %v", actionType, err)
		}
		if !execution.DryRun || execution.Status != types.ActionExecutionCompleted {
			t.Fatalf("unexpected execution: %+v", execution)
		}
		if execution.Result["dryRun"] != true {
			t.Fatalf("expected dry run result: %+v", execution.Result)
		}
	}
	if len(executor.Executions()) != 4 {
		t.Fatalf("expected 4 executions, got %d", len(executor.Executions()))
	}
}

func TestMockExecutorRejectsUnsupportedAction(t *testing.T) {
	executor := NewMockExecutor()
	_, err := executor.Execute(context.Background(), types.ActionRequest{
		ID:        "action-1",
		Type:      "unsafe",
		Risk:      types.ActionRiskLow,
		Status:    types.ActionStatusAllowed,
		Source:    "test",
		CreatedAt: time.Now(),
	})
	if err == nil {
		t.Fatal("expected unsupported action error")
	}
}

func TestEngineExecuteActionUsesConfiguredExecutor(t *testing.T) {
	engine, err := NewEngine(nil, nil)
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}
	if _, err := engine.ExecuteAction(context.Background(), types.ActionRequest{}); err == nil {
		t.Fatal("expected missing executor error")
	}

	engine.SetExecutor(NewMockExecutor())
	execution, err := engine.ExecuteAction(context.Background(), types.ActionRequest{
		ID:        "action-1",
		Type:      types.WorkflowActionEmitEvent,
		Risk:      types.ActionRiskLow,
		Status:    types.ActionStatusAllowed,
		Source:    "test",
		CreatedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("execute action: %v", err)
	}
	if execution.Type != types.WorkflowActionEmitEvent || !execution.DryRun {
		t.Fatalf("unexpected execution: %+v", execution)
	}
}
