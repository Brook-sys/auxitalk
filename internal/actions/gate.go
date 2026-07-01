package actions

import (
	"errors"
	"fmt"

	"github.com/Brook-sys/auxitalk/internal/config"
	"github.com/Brook-sys/auxitalk/pkg/types"
)

var (
	ErrActionDenied = errors.New("action denied by gate")
)

type Gate struct {
	mode config.Mode
}

func NewGate(mode config.Mode) *Gate {
	return &Gate{mode: mode}
}

func (g *Gate) Decide(action types.ActionRequest) (types.ActionStatus, error) {
	if err := action.Validate(); err != nil {
		return "", fmt.Errorf("invalid action: %w", err)
	}

	switch g.mode {
	case config.ModeDev:
		return g.decideDev(action)
	case config.ModeLocal:
		return g.decideLocal(action)
	case config.ModeStrict:
		return g.decideStrict(action)
	default:
		return "", fmt.Errorf("unknown runtime mode: %s", g.mode)
	}
}

func (g *Gate) decideDev(action types.ActionRequest) (types.ActionStatus, error) {
	return types.ActionStatusAllowed, nil
}

func (g *Gate) decideLocal(action types.ActionRequest) (types.ActionStatus, error) {
	switch action.Risk {
	case types.ActionRiskLow:
		return types.ActionStatusAllowed, nil
	case types.ActionRiskMedium:
		return types.ActionStatusConfirmed, nil
	case types.ActionRiskHigh:
		return types.ActionStatusDenied, ErrActionDenied
	default:
		return types.ActionStatusDenied, ErrActionDenied
	}
}

func (g *Gate) decideStrict(action types.ActionRequest) (types.ActionStatus, error) {
	switch action.Risk {
	case types.ActionRiskLow:
		return types.ActionStatusConfirmed, nil
	case types.ActionRiskMedium, types.ActionRiskHigh:
		return types.ActionStatusDenied, ErrActionDenied
	default:
		return types.ActionStatusDenied, ErrActionDenied
	}
}
