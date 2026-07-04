package types

import "testing"

func TestWorkflowRuleValidateAndMatches(t *testing.T) {
	rule := WorkflowRule{
		ID:      "reply-to-whatsapp",
		Enabled: true,
		Trigger: WorkflowTrigger{EventType: "message.received", Source: "whatsapp"},
		Actions: []WorkflowAction{{Type: "message.reply", Risk: ActionRiskMedium}},
	}
	if err := rule.Validate(); err != nil {
		t.Fatalf("expected valid rule: %v", err)
	}
	if !rule.Matches(Event{Type: "message.received", Source: "whatsapp"}, Session{}) {
		t.Fatal("expected rule to match")
	}
	if rule.Matches(Event{Type: "message.received", Source: "telegram"}, Session{}) {
		t.Fatal("expected source mismatch")
	}

	rule.Actions[0].Risk = "unsafe"
	if err := rule.Validate(); err == nil {
		t.Fatal("expected invalid risk error")
	}
}

func TestWorkflowConditionMatches(t *testing.T) {
	event := Event{
		SessionID: "session-1",
		Payload: map[string]any{
			"text":  "hello world",
			"count": 42,
		},
	}
	tests := []struct {
		cond  WorkflowCondition
		match bool
	}{
		{WorkflowCondition{Field: "sessionId", Operator: "==", Value: "session-1"}, true},
		{WorkflowCondition{Field: "payload.text", Operator: "equals", Value: "hello world"}, true},
		{WorkflowCondition{Field: "payload.text", Operator: "contains", Value: "world"}, true},
		{WorkflowCondition{Field: "payload.text", Operator: "starts_with", Value: "hello"}, true},
		{WorkflowCondition{Field: "payload.count", Operator: "==", Value: "42"}, true},
		{WorkflowCondition{Field: "payload.text", Operator: "!=", Value: "bye"}, true},
		{WorkflowCondition{Field: "payload.text", Operator: "not_contains", Value: "bye"}, true},
		{WorkflowCondition{Field: "payload.missing", Operator: "==", Value: ""}, true},
		{WorkflowCondition{Field: "payload.text", Operator: "==", Value: "wrong"}, false},
	}
	for _, tc := range tests {
		if got := tc.cond.Matches(event, Session{}); got != tc.match {
			t.Errorf("cond %+v: expected %v, got %v", tc.cond, tc.match, got)
		}
	}
}
