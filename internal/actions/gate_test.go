package actions

import (
	"errors"
	"testing"
	"time"

	"github.com/Brook-sys/auxitalk/internal/config"
	"github.com/Brook-sys/auxitalk/pkg/types"
)

func makeAction(risk types.ActionRisk) types.ActionRequest {
	return types.ActionRequest{
		ID:        "a1",
		Type:      "message.send",
		Risk:      risk,
		Status:    types.ActionStatusRequested,
		Source:    "test",
		CreatedAt: time.Now(),
	}
}

func TestGateDevMode(t *testing.T) {
	gate := NewGate(config.ModeDev)

	for _, risk := range []types.ActionRisk{types.ActionRiskLow, types.ActionRiskMedium, types.ActionRiskHigh} {
		status, err := gate.Decide(makeAction(risk))
		if err != nil {
			t.Fatalf("dev mode should allow %s: %v", risk, err)
		}
		if status != types.ActionStatusAllowed {
			t.Fatalf("expected allowed, got %s", status)
		}
	}
}

func TestGateLocalMode(t *testing.T) {
	gate := NewGate(config.ModeLocal)

	low, _ := gate.Decide(makeAction(types.ActionRiskLow))
	if low != types.ActionStatusAllowed {
		t.Fatalf("low should be allowed")
	}

	medium, _ := gate.Decide(makeAction(types.ActionRiskMedium))
	if medium != types.ActionStatusConfirmed {
		t.Fatalf("medium should require confirmation")
	}

	_, err := gate.Decide(makeAction(types.ActionRiskHigh))
	if !errors.Is(err, ErrActionDenied) {
		t.Fatalf("high should be denied in local mode")
	}
}

func TestGateStrictMode(t *testing.T) {
	gate := NewGate(config.ModeStrict)

	low, _ := gate.Decide(makeAction(types.ActionRiskLow))
	if low != types.ActionStatusConfirmed {
		t.Fatalf("low should require confirmation in strict")
	}

	_, err := gate.Decide(makeAction(types.ActionRiskMedium))
	if !errors.Is(err, ErrActionDenied) {
		t.Fatalf("medium should be denied in strict mode")
	}
}
