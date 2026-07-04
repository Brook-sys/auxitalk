package types

import (
	"errors"
	"fmt"
	"strings"
)

type WorkflowCondition struct {
	Field    string `json:"field"`
	Operator string `json:"operator"`
	Value    string `json:"value"`
}

type WorkflowTrigger struct {
	EventType  string              `json:"eventType"`
	Source     string              `json:"source,omitempty"`
	Conditions []WorkflowCondition `json:"conditions,omitempty"`
}

type WorkflowAction struct {
	Type    string         `json:"type"`
	Risk    ActionRisk     `json:"risk"`
	Payload map[string]any `json:"payload,omitempty"`
}

type Workflow struct {
	ID      string         `json:"id"`
	Name    string         `json:"name,omitempty"`
	Enabled bool           `json:"enabled"`
	Rules   []WorkflowRule `json:"rules"`
}

type WorkflowRule struct {
	ID      string           `json:"id"`
	Name    string           `json:"name,omitempty"`
	Enabled bool             `json:"enabled"`
	Trigger WorkflowTrigger  `json:"trigger"`
	Action  *WorkflowAction  `json:"action,omitempty"` // Deprecated, use Actions
	Actions []WorkflowAction `json:"actions,omitempty"`
}

func (w Workflow) Validate() error {
	if strings.TrimSpace(w.ID) == "" {
		return errors.New("workflow id is required")
	}
	if len(w.Rules) == 0 {
		return errors.New("workflow requires at least one rule")
	}
	seen := map[string]struct{}{}
	for _, rule := range w.Rules {
		if err := rule.Validate(); err != nil {
			return err
		}
		if _, exists := seen[rule.ID]; exists {
			return errors.New("workflow rule id must be unique")
		}
		seen[rule.ID] = struct{}{}
	}
	return nil
}

func (w Workflow) EnabledRules() []WorkflowRule {
	if !w.Enabled {
		return nil
	}
	rules := make([]WorkflowRule, 0, len(w.Rules))
	for _, rule := range w.Rules {
		if rule.Enabled {
			rules = append(rules, rule)
		}
	}
	return rules
}

func (r WorkflowRule) Validate() error {
	if strings.TrimSpace(r.ID) == "" {
		return errors.New("workflow rule id is required")
	}
	if strings.TrimSpace(r.Trigger.EventType) == "" {
		return errors.New("workflow trigger eventType is required")
	}
	actions := r.GetActions()
	if len(actions) == 0 {
		return errors.New("workflow requires at least one action")
	}
	for _, action := range actions {
		if strings.TrimSpace(action.Type) == "" {
			return errors.New("workflow action type is required")
		}
		switch action.Risk {
		case ActionRiskLow, ActionRiskMedium, ActionRiskHigh:
		default:
			return errors.New("workflow action risk is invalid")
		}
	}
	return nil
}

func (r WorkflowRule) GetActions() []WorkflowAction {
	if len(r.Actions) > 0 {
		return r.Actions
	}
	if r.Action != nil {
		return []WorkflowAction{*r.Action}
	}
	return nil
}

func (r WorkflowRule) Matches(event Event, session Session) bool {
	if !r.Enabled {
		return false
	}
	if r.Trigger.EventType != "*" && r.Trigger.EventType != event.Type {
		return false
	}
	if r.Trigger.Source != "" && r.Trigger.Source != event.Source {
		return false
	}
	for _, cond := range r.Trigger.Conditions {
		if !cond.Matches(event, session) {
			return false
		}
	}
	return true
}

func (c WorkflowCondition) Matches(event Event, session Session) bool {
	var actual string
	if c.Field == "sessionId" || c.Field == "session.id" {
		actual = event.SessionID
	} else if c.Field == "session.channel" {
		actual = session.Channel
	} else if c.Field == "session.state" {
		actual = session.State
	} else if strings.HasPrefix(c.Field, "session.metadata.") {
		key := strings.TrimPrefix(c.Field, "session.metadata.")
		if val, ok := session.Metadata[key]; ok {
			actual = fmt.Sprint(val)
		}
	} else if strings.HasPrefix(c.Field, "payload.") {
		key := strings.TrimPrefix(c.Field, "payload.")
		if val, ok := event.Payload[key]; ok {
			actual = fmt.Sprint(val)
		}
	} else {
		return false // unsupported field
	}

	switch c.Operator {
	case "equals", "==":
		return actual == c.Value
	case "not_equals", "!=":
		return actual != c.Value
	case "contains":
		return strings.Contains(actual, c.Value)
	case "not_contains":
		return !strings.Contains(actual, c.Value)
	case "starts_with":
		return strings.HasPrefix(actual, c.Value)
	case "ends_with":
		return strings.HasSuffix(actual, c.Value)
	default:
		return false
	}
}
