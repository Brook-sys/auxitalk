package types

import (
	"testing"
	"time"
)

func TestActionExecutionValidate(t *testing.T) {
	execution := ActionExecution{
		ID:        "exec-1",
		ActionID:  "action-1",
		Type:      WorkflowActionRunCommand,
		Status:    ActionExecutionCompleted,
		DryRun:    true,
		CreatedAt: time.Now(),
	}
	if err := execution.Validate(); err != nil {
		t.Fatalf("expected valid execution: %v", err)
	}

	execution.Status = "unknown"
	if err := execution.Validate(); err == nil {
		t.Fatal("expected invalid status error")
	}
}

func TestIsWorkflowActionType(t *testing.T) {
	for _, actionType := range []string{WorkflowActionSendMessage, WorkflowActionRunCommand, WorkflowActionCallPlugin, WorkflowActionEmitEvent} {
		if !IsWorkflowActionType(actionType) {
			t.Fatalf("expected supported action type %s", actionType)
		}
	}
	if IsWorkflowActionType("dangerous") {
		t.Fatal("expected unsupported action type")
	}
}
