package types

import "testing"

func TestWorkflowRuleValidateAndMatches(t *testing.T) {
	rule := WorkflowRule{
		ID:      "reply-to-whatsapp",
		Enabled: true,
		Trigger: WorkflowTrigger{EventType: "message.received", Source: "whatsapp"},
		Action:  WorkflowAction{Type: "message.reply", Risk: ActionRiskMedium},
	}
	if err := rule.Validate(); err != nil {
		t.Fatalf("expected valid rule: %v", err)
	}
	if !rule.Matches(Event{Type: "message.received", Source: "whatsapp"}) {
		t.Fatal("expected rule to match")
	}
	if rule.Matches(Event{Type: "message.received", Source: "telegram"}) {
		t.Fatal("expected source mismatch")
	}

	rule.Action.Risk = "unsafe"
	if err := rule.Validate(); err == nil {
		t.Fatal("expected invalid risk error")
	}
}
