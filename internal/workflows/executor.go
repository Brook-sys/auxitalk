package workflows

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Brook-sys/auxitalk/pkg/types"
)

type Executor interface {
	Execute(context.Context, types.ActionRequest) (types.ActionExecution, error)
}

type MockExecutor struct {
	mu         sync.RWMutex
	now        func() time.Time
	seq        uint64
	executions []types.ActionExecution
}

func NewMockExecutor() *MockExecutor {
	return &MockExecutor{now: func() time.Time { return time.Now().UTC() }}
}

func (e *MockExecutor) Execute(ctx context.Context, action types.ActionRequest) (types.ActionExecution, error) {
	if err := action.Validate(); err != nil {
		return types.ActionExecution{}, err
	}
	if !types.IsWorkflowActionType(action.Type) {
		return types.ActionExecution{}, fmt.Errorf("unsupported workflow action type: %s", action.Type)
	}
	if err := ctx.Err(); err != nil {
		return types.ActionExecution{}, err
	}

	execution := e.newExecution(action)
	if err := execution.Validate(); err != nil {
		return types.ActionExecution{}, err
	}
	e.mu.Lock()
	e.executions = append(e.executions, execution)
	e.mu.Unlock()
	return execution, nil
}

func (e *MockExecutor) Executions() []types.ActionExecution {
	e.mu.RLock()
	defer e.mu.RUnlock()
	executions := make([]types.ActionExecution, len(e.executions))
	copy(executions, e.executions)
	return executions
}

func (e *MockExecutor) newExecution(action types.ActionRequest) types.ActionExecution {
	e.mu.Lock()
	e.seq++
	seq := e.seq
	now := e.now()
	e.mu.Unlock()

	return types.ActionExecution{
		ID:          fmt.Sprintf("mock-exec-%d", seq),
		ActionID:    action.ID,
		Type:        action.Type,
		Status:      types.ActionExecutionCompleted,
		DryRun:      true,
		Input:       cloneMap(action.Payload),
		Result:      mockResult(action),
		CreatedAt:   now,
		CompletedAt: now,
	}
}

func mockResult(action types.ActionRequest) map[string]any {
	result := map[string]any{
		"dryRun": true,
		"type":   action.Type,
	}
	switch action.Type {
	case types.WorkflowActionSendMessage:
		result["messageQueued"] = false
		result["description"] = "message send simulated"
	case types.WorkflowActionRunCommand:
		result["commandExecuted"] = false
		result["description"] = "command execution simulated"
	case types.WorkflowActionCallPlugin:
		result["pluginCalled"] = false
		result["description"] = "plugin call simulated"
	case types.WorkflowActionEmitEvent:
		result["eventEmitted"] = false
		result["description"] = "event emit simulated"
	}
	return result
}

func cloneMap(input map[string]any) map[string]any {
	if input == nil {
		return nil
	}
	output := make(map[string]any, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}
