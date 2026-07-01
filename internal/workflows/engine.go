package workflows

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Brook-sys/auxitalk/pkg/types"
)

type ActionSink interface {
	RequestAction(context.Context, types.ActionRequest) error
}

type Engine struct {
	mu       sync.RWMutex
	rules    []types.WorkflowRule
	sink     ActionSink
	executor Executor
	now      func() time.Time
	idSeq    uint64
}

func NewEngine(sink ActionSink, rules []types.WorkflowRule) (*Engine, error) {
	engine := &Engine{
		sink: sink,
		now:  func() time.Time { return time.Now().UTC() },
	}
	if err := engine.SetRules(rules); err != nil {
		return nil, err
	}
	return engine, nil
}

func (e *Engine) SetRules(rules []types.WorkflowRule) error {
	validated := make([]types.WorkflowRule, 0, len(rules))
	for _, rule := range rules {
		if err := rule.Validate(); err != nil {
			return err
		}
		validated = append(validated, rule)
	}

	e.mu.Lock()
	defer e.mu.Unlock()
	e.rules = validated
	return nil
}

func (e *Engine) Rules() []types.WorkflowRule {
	e.mu.RLock()
	defer e.mu.RUnlock()
	rules := make([]types.WorkflowRule, len(e.rules))
	copy(rules, e.rules)
	return rules
}

func (e *Engine) SetExecutor(executor Executor) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.executor = executor
}

func (e *Engine) ExecuteAction(ctx context.Context, action types.ActionRequest) (types.ActionExecution, error) {
	e.mu.RLock()
	executor := e.executor
	e.mu.RUnlock()
	if executor == nil {
		return types.ActionExecution{}, fmt.Errorf("workflow executor is not configured")
	}
	return executor.Execute(ctx, action)
}

func (e *Engine) HandleEvent(ctx context.Context, event types.Event) ([]types.ActionRequest, error) {
	if err := event.Validate(); err != nil {
		return nil, err
	}

	e.mu.RLock()
	rules := make([]types.WorkflowRule, len(e.rules))
	copy(rules, e.rules)
	e.mu.RUnlock()

	requested := []types.ActionRequest{}
	for _, rule := range rules {
		if !rule.Matches(event) {
			continue
		}
		action := e.newAction(rule, event)
		if err := action.Validate(); err != nil {
			return requested, err
		}
		if e.sink != nil {
			if err := e.sink.RequestAction(ctx, action); err != nil {
				return requested, err
			}
		}
		requested = append(requested, action)
	}
	return requested, nil
}

func (e *Engine) newAction(rule types.WorkflowRule, event types.Event) types.ActionRequest {
	e.mu.Lock()
	e.idSeq++
	seq := e.idSeq
	now := e.now()
	e.mu.Unlock()

	payload := map[string]any{
		"workflowRuleId": rule.ID,
		"eventId":        event.ID,
		"eventType":      event.Type,
		"eventSource":    event.Source,
	}
	for key, value := range rule.Action.Payload {
		payload[key] = value
	}

	return types.ActionRequest{
		ID:        fmt.Sprintf("workflow-%s-%d", rule.ID, seq),
		Type:      rule.Action.Type,
		Risk:      rule.Action.Risk,
		Status:    types.ActionStatusRequested,
		Source:    "workflow:" + rule.ID,
		SessionID: event.SessionID,
		Payload:   payload,
		CreatedAt: now,
	}
}
