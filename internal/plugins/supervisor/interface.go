package supervisor

import (
	"context"
	"encoding/json"
)

type SupervisorInterface interface {
	Register(spec ProcessSpec) error
	Start(ctx context.Context, id string) error
	Stop(id string) error
	Call(ctx context.Context, id string, method string, params any) (json.RawMessage, error)
	IsRunning(id string) bool
	List() []string
}
