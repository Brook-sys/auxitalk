package workflows

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Brook-sys/auxitalk/pkg/types"
)

type ActionSink interface {
	RequestAction(context.Context, types.ActionRequest) error
}

type SessionResolver interface {
	Get(id string) (types.Session, error)
}

type Engine struct {
	mu       sync.RWMutex
	rules    []types.WorkflowRule
	sink     ActionSink
	executor Executor
	sessions SessionResolver
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

func (e *Engine) SetSessions(resolver SessionResolver) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.sessions = resolver
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
	fmt.Printf("[workflow] handle event type=%s source=%s rules=%d\n", event.Type, event.Source, len(rules))
	for _, rule := range rules {
		if !rule.Matches(event) {
			continue
		}
		for _, reqAction := range e.newActions(rule, event) {
			if err := reqAction.Validate(); err != nil {
				return requested, err
			}
			if e.sink != nil {
				if err := e.sink.RequestAction(ctx, reqAction); err != nil {
					return requested, err
				}
			}
			requested = append(requested, reqAction)
		}
	}
	return requested, nil
}

func (e *Engine) newActions(rule types.WorkflowRule, event types.Event) []types.ActionRequest {
	var session types.Session
	e.mu.RLock()
	resolver := e.sessions
	e.mu.RUnlock()
	if resolver != nil && event.SessionID != "" {
		if s, err := resolver.Get(event.SessionID); err == nil {
			session = s
		}
	}

	actions := rule.GetActions()
	requests := make([]types.ActionRequest, 0, len(actions))
	for _, action := range actions {
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
		for key, value := range action.Payload {
			payload[key] = interpolate(value, event, session)
		}

		requests = append(requests, types.ActionRequest{
			ID:        fmt.Sprintf("workflow-%s-%d", rule.ID, seq),
			Type:      action.Type,
			Risk:      action.Risk,
			Status:    types.ActionStatusRequested,
			Source:    "workflow:" + rule.ID,
			SessionID: event.SessionID,
			Payload:   payload,
			CreatedAt: now,
		})
	}
	return requests
}

func interpolate(value any, event types.Event, session types.Session) any {
	switch v := value.(type) {
	case string:
		return interpolateString(v, event, session)
	case map[string]any:
		out := make(map[string]any, len(v))
		for key, item := range v {
			out[key] = interpolate(item, event, session)
		}
		return out
	case []any:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = interpolate(item, event, session)
		}
		return out
	default:
		return value
	}
}

func interpolateString(str string, event types.Event, session types.Session) string {
	str = strings.ReplaceAll(str, "{{event.id}}", event.ID)
	str = strings.ReplaceAll(str, "{{event.type}}", event.Type)
	str = strings.ReplaceAll(str, "{{event.source}}", event.Source)
	str = strings.ReplaceAll(str, "{{event.sessionId}}", event.SessionID)
	for key, value := range event.Payload {
		str = strings.ReplaceAll(str, "{{payload."+key+"}}", fmt.Sprint(value))
		str = strings.ReplaceAll(str, "{{event.payload."+key+"}}", fmt.Sprint(value))
	}
	str = strings.ReplaceAll(str, "{{session.id}}", session.ID)
	str = strings.ReplaceAll(str, "{{session.state}}", session.State)
	for key, value := range session.Metadata {
		str = strings.ReplaceAll(str, "{{session.metadata."+key+"}}", fmt.Sprint(value))
	}
	return str
}
