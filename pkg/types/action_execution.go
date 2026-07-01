package types

import (
	"errors"
	"strings"
	"time"
)

const (
	WorkflowActionSendMessage = "send_message"
	WorkflowActionRunCommand  = "run_command"
	WorkflowActionCallPlugin  = "call_plugin"
	WorkflowActionEmitEvent   = "emit_event"
)

type ActionExecutionStatus string

const (
	ActionExecutionCompleted ActionExecutionStatus = "completed"
	ActionExecutionFailed    ActionExecutionStatus = "failed"
	ActionExecutionSkipped   ActionExecutionStatus = "skipped"
)

type ActionExecution struct {
	ID          string                `json:"id"`
	ActionID    string                `json:"actionId"`
	Type        string                `json:"type"`
	Status      ActionExecutionStatus `json:"status"`
	DryRun      bool                  `json:"dryRun"`
	Input       map[string]any        `json:"input,omitempty"`
	Result      map[string]any        `json:"result,omitempty"`
	Error       string                `json:"error,omitempty"`
	CreatedAt   time.Time             `json:"createdAt"`
	CompletedAt time.Time             `json:"completedAt,omitempty"`
}

func (e ActionExecution) Validate() error {
	if strings.TrimSpace(e.ID) == "" {
		return errors.New("action execution id is required")
	}
	if strings.TrimSpace(e.ActionID) == "" {
		return errors.New("action execution actionId is required")
	}
	if strings.TrimSpace(e.Type) == "" {
		return errors.New("action execution type is required")
	}
	switch e.Status {
	case ActionExecutionCompleted, ActionExecutionFailed, ActionExecutionSkipped:
	default:
		return errors.New("action execution status is invalid")
	}
	if e.CreatedAt.IsZero() {
		return errors.New("action execution createdAt is required")
	}
	return nil
}

func IsWorkflowActionType(actionType string) bool {
	switch actionType {
	case WorkflowActionSendMessage, WorkflowActionRunCommand, WorkflowActionCallPlugin, WorkflowActionEmitEvent:
		return true
	default:
		return false
	}
}
