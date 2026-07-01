package supervisor

import (
	"context"
	"os/exec"
)

type SupervisorInterface interface {
	Register(id string, cmd *exec.Cmd) error
	Start(ctx context.Context, id string) error
	Stop(id string) error
	IsRunning(id string) bool
	List() []string
}
