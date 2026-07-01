package types

import (
	"errors"
	"strings"
)

type WorkflowTrigger struct {
	EventType string `json:"eventType"`
	Source    string `json:"source,omitempty"`
}

type WorkflowAction struct {
	Type    string         `json:"type"`
	Risk    ActionRisk     `json:"risk"`
	Payload map[string]any `json:"payload,omitempty"`
}

type WorkflowRule struct {
	ID      string          `json:"id"`
	Name    string          `json:"name,omitempty"`
	Enabled bool            `json:"enabled"`
	Trigger WorkflowTrigger `json:"trigger"`
	Action  WorkflowAction  `json:"action"`
}

func (r WorkflowRule) Validate() error {
	if strings.TrimSpace(r.ID) == "" {
		return errors.New("workflow rule id is required")
	}
	if strings.TrimSpace(r.Trigger.EventType) == "" {
		return errors.New("workflow trigger eventType is required")
	}
	if strings.TrimSpace(r.Action.Type) == "" {
		return errors.New("workflow action type is required")
	}
	switch r.Action.Risk {
	case ActionRiskLow, ActionRiskMedium, ActionRiskHigh:
	default:
		return errors.New("workflow action risk is invalid")
	}
	return nil
}

func (r WorkflowRule) Matches(event Event) bool {
	if !r.Enabled {
		return false
	}
	if r.Trigger.EventType != "*" && r.Trigger.EventType != event.Type {
		return false
	}
	if r.Trigger.Source != "" && r.Trigger.Source != event.Source {
		return false
	}
	return true
}
